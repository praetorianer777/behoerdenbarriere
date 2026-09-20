package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
)

// fakeScanner stands in for the browser: it answers from a link map, so the crawl
// logic can be tested without Chrome.
type fakeScanner struct {
	mu      sync.Mutex
	links   map[string][]string
	visited []string
	delay   time.Duration
	broken  map[string]bool
}

func (f *fakeScanner) Scan(ctx context.Context, url string) scanner.PageScan {
	f.mu.Lock()
	f.visited = append(f.visited, url)
	links := f.links[url]
	broken := f.broken[url]
	f.mu.Unlock()

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
		}
	}
	if broken {
		return scanner.PageScan{Result: model.PageResult{URL: url, Err: "timeout"}}
	}
	return scanner.PageScan{
		Result: model.PageResult{URL: url, DOMNodes: 500, HTTPStatus: 200},
		Links:  links,
	}
}

func (f *fakeScanner) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.visited...)
}

func robotsServer(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			if body == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestCrawlFollowsInternalLinksOnly(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{links: map[string][]string{
		base + "/": {
			base + "/kontakt",
			base + "/presse",
			"https://twitter.com/amt",
			base + "/formular.pdf",
		},
		base + "/kontakt": {base + "/"},
	}}

	outcome, err := New(fake, Config{MaxPages: 10, MaxDepth: 2, RatePerSec: 1000}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	urls := map[string]model.PageResult{}
	for _, p := range got {
		urls[p.URL] = p
	}
	if len(urls) != 3 {
		t.Fatalf("expected 3 pages, got %v", keysOf(urls))
	}
	for _, want := range []string{base + "/", base + "/kontakt", base + "/presse"} {
		if _, ok := urls[want]; !ok {
			t.Errorf("%s not checked", want)
		}
	}
	if !urls[base+"/"].IsEntry {
		t.Error("start page not marked as entry page")
	}
	if !urls[base+"/kontakt"].Priority {
		t.Error("contact page not marked as a priority page")
	}
	if urls[base+"/presse"].Priority {
		t.Error("press page wrongly marked as a priority page")
	}
	if urls[base+"/kontakt"].Depth != 1 {
		t.Errorf("depth = %d, want 1", urls[base+"/kontakt"].Depth)
	}
}

func TestCrawlRespectsPageBudget(t *testing.T) {
	base := robotsServer(t, "")
	links := make([]string, 0, 20)
	for i := range 20 {
		links = append(links, fmt.Sprintf("%s/seite-%d", base, i))
	}
	fake := &fakeScanner{links: map[string][]string{base + "/": links}}

	outcome, err := New(fake, Config{MaxPages: 5, MaxDepth: 3, RatePerSec: 1000}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("checked %d pages, want 5", len(got))
	}
}

// The budget has to go to the pages that matter legally, not to whatever the
// navigation happens to list first.
func TestCrawlSpendsBudgetOnPriorityPages(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{links: map[string][]string{
		base + "/": {
			base + "/presse/eins", base + "/presse/zwei", base + "/presse/drei",
			base + "/erklaerung-zur-barrierefreiheit", base + "/kontakt",
		},
	}}

	outcome, err := New(fake, Config{MaxPages: 3, MaxDepth: 2, RatePerSec: 1000}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	checked := map[string]bool{}
	for _, p := range got {
		checked[p.URL] = true
	}
	for _, want := range []string{base + "/erklaerung-zur-barrierefreiheit", base + "/kontakt"} {
		if !checked[want] {
			t.Errorf("%s was not checked although the budget was enough", want)
		}
	}
}

func TestCrawlRespectsDepth(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{links: map[string][]string{
		base + "/":      {base + "/a"},
		base + "/a":     {base + "/a/b"},
		base + "/a/b":   {base + "/a/b/c"},
		base + "/a/b/c": {},
	}}

	outcome, err := New(fake, Config{MaxPages: 50, MaxDepth: 1, RatePerSec: 1000}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("checked %d pages, want 2 (start page plus one level)", len(got))
	}
}

func TestCrawlObeysRobotsDisallow(t *testing.T) {
	base := robotsServer(t, "User-agent: *\nDisallow: /intern\n")
	fake := &fakeScanner{links: map[string][]string{
		base + "/": {base + "/intern/geheim", base + "/kontakt"},
	}}

	outcome, err := New(fake, Config{MaxPages: 10, MaxDepth: 2, RatePerSec: 1000}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	for _, p := range got {
		if strings.Contains(p.URL, "/intern") {
			t.Fatalf("a forbidden page was fetched: %s", p.URL)
		}
	}
	if len(got) != 2 {
		t.Fatalf("checked %d pages, want 2", len(got))
	}
}

// If the start page itself is forbidden, nothing is checked — following links from a
// site we were not allowed to enter would be worse than not checking it.
func TestCrawlStopsWhenStartPageForbidden(t *testing.T) {
	base := robotsServer(t, "User-agent: *\nDisallow: /\n")
	fake := &fakeScanner{}

	_, err := New(fake, Config{MaxPages: 10, RatePerSec: 1000}).Crawl(context.Background(), base)
	if !errors.Is(err, ErrDisallowed) {
		t.Fatalf("err = %v, want ErrDisallowed", err)
	}
	if len(fake.seen()) != 0 {
		t.Fatalf("pages were fetched anyway: %v", fake.seen())
	}
}

// A missing robots.txt is permission. A server that is briefly unavailable must not
// silently drop an authority out of the ranking.
func TestCrawlTreatsMissingRobotsAsAllowed(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{links: map[string][]string{base + "/": {}}}

	outcome, err := New(fake, Config{MaxPages: 5, RatePerSec: 1000}).Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("checked %d pages, want 1", len(got))
	}
}

func TestCrawlKeepsRateLimit(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{links: map[string][]string{
		base + "/": {base + "/a", base + "/b"},
	}}

	started := time.Now()
	outcome, err := New(fake, Config{MaxPages: 3, MaxDepth: 1, RatePerSec: 20}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("checked %d pages, want 3", len(got))
	}
	// Three pages at 20 per second take at least two intervals of 50 ms.
	if elapsed := time.Since(started); elapsed < 100*time.Millisecond {
		t.Fatalf("three pages in %v — the rate limit is not being kept", elapsed)
	}
}

// A page that cannot be loaded belongs in the result as a failure, so the scan can
// report how much of the site could be checked at all.
func TestCrawlKeepsGoingAfterAFailedPage(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{
		links:  map[string][]string{base + "/": {base + "/kaputt", base + "/gut"}},
		broken: map[string]bool{base + "/kaputt": true},
	}

	outcome, err := New(fake, Config{MaxPages: 10, MaxDepth: 1, RatePerSec: 1000}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("checked %d pages, want 3", len(got))
	}
	var failed int
	for _, p := range got {
		if p.Failed() {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("%d failed pages, want 1", failed)
	}
}

// Running into the time budget yields what was checked so far, not an error: a partial
// result still says something about the site.
func TestCrawlReturnsPartialResultOnTimeout(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{
		links: map[string][]string{base + "/": {base + "/a", base + "/b", base + "/c"}},
		delay: 80 * time.Millisecond,
	}

	outcome, err := New(fake, Config{MaxPages: 10, MaxDepth: 1, RatePerSec: 1000, Timeout: 150 * time.Millisecond}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if len(got) == 0 || len(got) > 3 {
		t.Fatalf("checked %d pages — expected a partial result", len(got))
	}
}

// A cancellation from the outside is a shutdown, and the caller has to hear about it.
func TestCrawlReportsOutsideCancellation(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{links: map[string][]string{base + "/": {base + "/a"}}, delay: 50 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := New(fake, Config{MaxPages: 10, MaxDepth: 1, RatePerSec: 1000}).Crawl(ctx, base)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestCrawlRejectsUnusableStartURL(t *testing.T) {
	if _, err := New(&fakeScanner{}, Config{}).Crawl(context.Background(), "nicht-mal-eine-url"); err == nil {
		t.Fatal("unusable start URL was accepted")
	}
}

func keysOf(m map[string]model.PageResult) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// A wrong "this authority has no accessibility statement" is an accusation, and it
// must not depend on how many pages the budget allowed. Three federal agencies that
// clearly link their statement from the start page were recorded as having none,
// because contact and imprint were fetched first.
func TestCrawlAlwaysReachesTheAccessibilityStatement(t *testing.T) {
	base := robotsServer(t, "")
	fake := &fakeScanner{links: map[string][]string{
		base + "/": {
			base + "/kontakt",
			base + "/impressum",
			base + "/suche",
			base + "/formulare",
			base + "/DE/Service/Barrierefreiheit/erklaerung-zur-barrierefreiheit.html",
		},
	}}

	// Two pages only: the start page and one more.
	outcome, err := New(fake, Config{MaxPages: 2, MaxDepth: 1, RatePerSec: 1000}).
		Crawl(context.Background(), base)
	got := outcome.Pages
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	var reached bool
	for _, page := range got {
		if strings.Contains(page.URL, "erklaerung-zur-barrierefreiheit") {
			reached = true
		}
	}
	if !reached {
		t.Fatalf("the statement was not reached: %v", urlsOf(got))
	}
}

func urlsOf(pages []model.PageResult) []string {
	out := make([]string, 0, len(pages))
	for _, page := range pages {
		out = append(out, page.URL)
	}
	return out
}

// Zwei Prüfungen laufen gleichzeitig durch denselben Crawler. Für Behörden unter
// demselben Host — jedes Bundesministerium liegt unter bund.de — muss der Takt dann
// trotzdem gelten, sonst wird aus einer Anfrage pro Sekunde eine je laufender Prüfung.
func TestConcurrentCrawlsShareTheHostBudget(t *testing.T) {
	scanner := &fakeScanner{links: map[string][]string{}}
	interval := 80 * time.Millisecond
	c := New(scanner, Config{
		MaxPages:   2,
		MaxDepth:   1,
		RatePerSec: float64(time.Second) / float64(interval),
		Timeout:    10 * time.Second,
	})

	srv := robotsServer(t, "User-agent: *\nAllow: /")

	started := time.Now()
	var wg sync.WaitGroup
	for _, path := range []string{"/amt-a", "/amt-b"} {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			if _, err := c.Crawl(context.Background(), srv+path); err != nil {
				t.Error(err)
			}
		}(path)
	}
	wg.Wait()

	// Zwei Einstiegsseiten auf demselben Host: eine sofort, die zweite erst nach dem
	// Intervall.
	if elapsed := time.Since(started); elapsed < interval {
		t.Errorf("two scans of one host took %v, want at least %v", elapsed, interval)
	}
	if got := len(scanner.seen()); got != 2 {
		t.Errorf("%d pages checked, want 2", got)
	}
}

// zoll.de und die übrigen Bundesbehörden verlangen in ihrer robots.txt 180 Sekunden
// Pause. Bei zehn Minuten Zeitbudget bleiben davon ein bis drei Seiten — und ein Score
// über eine Seite ist nicht der Score, den wir definieren.
func TestBudgetStretchesForSlowSites(t *testing.T) {
	normal := 10 * time.Minute

	if got := budget(normal, time.Second); got != normal {
		t.Errorf("schnelle Seite: %v, want %v", got, normal)
	}
	if got := budget(normal, 180*time.Second); got <= normal {
		t.Errorf("langsame Seite: %v, want mehr als %v", got, normal)
	}
	// Aber nicht unbegrenzt: Eine Behörde darf die Warteschlange nicht einen
	// Nachmittag lang blockieren.
	if got := budget(normal, time.Hour); got != slowSiteBudget {
		t.Errorf("sehr langsame Seite: %v, want %v", got, slowSiteBudget)
	}
}
