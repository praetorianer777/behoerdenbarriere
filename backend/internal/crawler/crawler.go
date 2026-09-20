// Package crawler walks an authority's website and has every page checked.
package crawler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"golang.org/x/time/rate"

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
}

func New(s PageScanner, cfg Config) *Crawler {
	return &Crawler{
		scanner: s,
		cfg:     cfg.withDefaults(),
		robots:  newRobotsCache(&http.Client{Timeout: 20 * time.Second}),
	}
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
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	// robots.txt may ask for a longer pause than our own rate limit; the stricter of
	// the two wins.
	interval := time.Duration(float64(time.Second) / c.cfg.RatePerSec)
	if delay := c.robots.CrawlDelay(ctx, start); delay > interval {
		interval = delay
	}
	limiter := rate.NewLimiter(rate.Every(interval), 1)

	f := newFrontier()
	f.push(Target{URL: start, Depth: 0, IsEntry: true})

	outcome := Outcome{Pages: make([]model.PageResult, 0, c.cfg.MaxPages)}
	seen := map[string]bool{}
	for len(outcome.Pages) < c.cfg.MaxPages && !f.empty() {
		target, ok := f.pop()
		if !ok {
			break
		}
		if err := limiter.Wait(ctx); err != nil {
			break
		}

		scan := c.scanner.Scan(ctx, target.URL)
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

	// Our own deadline is a budget, not a failure: what was checked up to then counts.
	// A cancellation from the outside is different — then the caller is shutting down
	// and wants to know.
	if err := parent.Err(); err != nil {
		return outcome, err
	}
	return outcome, nil
}
