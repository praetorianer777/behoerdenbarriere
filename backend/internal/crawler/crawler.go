// Package crawler walks an authority's website and has every page checked.
package crawler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
	"github.com/praetorianer777/behoerdenbarriere/internal/statement"
)

// PageScanner is the part of the scanner the crawler uses. An interface, so the
// crawl logic can be tested without a browser.
type PageScanner interface {
	Scan(ctx context.Context, url string) scanner.PageScan
}

type Config struct {
	MaxPages   int
	MaxDepth   int
	RatePerSec float64
	Timeout    time.Duration
}

func (c Config) withDefaults() Config {
	if c.MaxPages <= 0 {
		c.MaxPages = 100
	}
	if c.MaxDepth < 0 {
		c.MaxDepth = 0
	}
	if c.RatePerSec <= 0 {
		c.RatePerSec = 1
	}
	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Minute
	}
	return c
}

type Crawler struct {
	scanner PageScanner
	cfg     Config
	robots  *robotsCache
	hosts   *HostLimiter
}

func New(s PageScanner, cfg Config) *Crawler {
	return &Crawler{
		scanner: s,
		cfg:     cfg.withDefaults(),
		robots:  newRobotsCache(&http.Client{Timeout: 20 * time.Second}),
		hosts:   NewHostLimiter(),
	}
}

// WithHostLimiter shares one limiter between crawlers. Several scans running at once
// must not each get their own budget for the same server — see HostLimiter.
func (c *Crawler) WithHostLimiter(hosts *HostLimiter) *Crawler {
	if hosts != nil {
		c.hosts = hosts
	}
	return c
}

// ErrDisallowed means robots.txt forbids the start page. Then nothing is checked at
// all — following links from a page we were not allowed to fetch would be worse.
var ErrDisallowed = errors.New("robots.txt forbids the start page")

// Outcome is what a crawl leaves behind.
type Outcome struct {
	Pages []model.PageResult
	// SeenLinks are the addresses the crawl came across but did not necessarily
	// fetch. What was seen and not read is the difference between "the authority has
	// no accessibility statement" and "robots.txt did not let us look" — the RKI
	// excludes the very directory theirs sits in.
	SeenLinks []string
}

// slowSiteBudget caps what a single authority may take, however patient its robots.txt
// asks us to be. Half an hour is enough for ten pages at a three-minute delay, and
// little enough that one slow site does not block the queue for an afternoon.
const slowSiteBudget = 30 * time.Minute

// budget stretches the time for a site that asks for long pauses. The federal CMS ships
// robots.txt with Crawl-delay: 180, which we honour — and in the usual ten minutes that
// leaves three pages, sometimes one. A score over one page is not the score we define.
//
// Stretched, not waived: the pause between requests stays exactly as asked.
func budget(normal, interval time.Duration) time.Duration {
	const pagesWorthHaving = 10

	wanted := interval * pagesWorthHaving
	if wanted <= normal {
		return normal
	}
	return min(wanted, slowSiteBudget)
}

// Crawl walks the site from startURL and returns the result for every page checked.
// Running into the time or page budget is not an error: a partial result still says
// something, and every page is checked before it is counted.
func (c *Crawler) Crawl(ctx context.Context, startURL string) (Outcome, error) {
	start := Normalize(startURL)
	if start == "" {
		return Outcome{}, errors.New("unusable start URL: " + startURL)
	}
	if !c.robots.Allowed(ctx, start) {
		return Outcome{}, ErrDisallowed
	}

	parent := ctx

	// robots.txt may ask for a longer pause than our own rate limit; the stricter of
	// the two wins.
	interval := time.Duration(float64(time.Second) / c.cfg.RatePerSec)
	if delay := c.robots.CrawlDelay(ctx, start); delay > interval {
		interval = delay
	}

	// The budget is a point in time we check against, not a deadline on the context: a
	// site that turns out to have moved can be entitled to more time than the one we
	// set out for, and a context deadline cannot be moved once it is set.
	deadline := time.Now().Add(budget(c.cfg.Timeout, interval))

	f := newFrontier()
	f.push(Target{URL: start, Depth: 0, IsEntry: true})

	outcome := Outcome{Pages: make([]model.PageResult, 0, c.cfg.MaxPages)}
	seen := map[string]bool{}
	// Where we have actually been. The frontier keeps us from queueing the same
	// address twice, but a redirect lands on an address we may already have queued
	// under its own name.
	fetched := map[string]bool{}
	for len(outcome.Pages) < c.cfg.MaxPages && !f.empty() {
		target, ok := f.pop()
		if !ok {
			break
		}
		if fetched[target.URL] {
			continue
		}
		// Sitting out a three-minute pause for a page that is then over budget
		// anyway costs the authority a request and us the wait.
		if time.Now().Add(interval).After(deadline) {
			break
		}
		// Der Takt gilt dem Host, nicht dem Lauf: Eine nicht lesbare Adresse wäre
		// hier schon vorher aussortiert worden.
		host, _ := hostOf(target.URL)
		if err := c.hosts.Wait(ctx, host, interval); err != nil {
			break
		}

		requested := target.URL
		scan := c.scanner.Scan(ctx, requested)
		if final := Normalize(scan.FinalURL); final != "" && final != target.URL {
			// An authority that has moved keeps the old domain alive as a redirect.
			// The site we are allowed to walk is the one we ended up on; measured
			// against the address we asked for, every link on it looks foreign and
			// the crawl stops after the start page.
			if target.IsEntry && !SameSite(start, final) {
				start = final
				// The new host has its own robots.txt, and with it its own
				// pace and its own claim on our patience. Adopting the site
				// without adopting its rules would be the rude half of this.
				if delay := c.robots.CrawlDelay(ctx, start); delay > interval {
					interval = delay
					if until := time.Now().Add(budget(c.cfg.Timeout, interval)); until.After(deadline) {
						deadline = until
					}
				}
			}
			if SameSite(start, final) {
				target.URL = final
			}
		}
		fetched[requested] = true
		fetched[target.URL] = true
		scan.Result.URL = target.URL
		scan.Result.Depth = target.Depth
		scan.Result.IsEntry = target.IsEntry
		scan.Result.Priority = target.Priority
		outcome.Pages = append(outcome.Pages, scan.Result)

		if target.Depth >= c.cfg.MaxDepth {
			continue
		}
		for _, link := range scan.Links {
			normalized := Normalize(link)
			if normalized == "" || !SameSite(start, normalized) {
				continue
			}
			if !seen[normalized] {
				seen[normalized] = true
				outcome.SeenLinks = append(outcome.SeenLinks, normalized)
			}
			if !c.robots.Allowed(ctx, normalized) {
				continue
			}
			f.push(Target{
				URL:       normalized,
				Depth:     target.Depth + 1,
				Priority:  IsPriority(normalized),
				Statement: statement.IsStatementURL(normalized),
			})
		}
	}

	// Running out of budget is not a failure: what was checked up to then counts. A
	// cancellation from the outside is different — then the caller is shutting down
	// and wants to know.
	if err := parent.Err(); err != nil {
		return outcome, err
	}
	return outcome, nil
}
