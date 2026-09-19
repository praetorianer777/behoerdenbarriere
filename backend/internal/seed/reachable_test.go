package seed

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

// Checks whether every seeded start page actually answers. The test reaches out to
// around a hundred authorities, so it only runs on request (SEEDS_NETWORK_TEST=1) —
// but it is the only way a typo in a URL shows up before the first scan.
func TestSeedURLsAreReachable(t *testing.T) {
	if os.Getenv("SEEDS_NETWORK_TEST") == "" {
		t.Skip("SEEDS_NETWORK_TEST not set")
	}
	agencies, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup

	for _, a := range agencies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			// Some authorities answer HEAD with 405 but serve GET fine, so GET is what
			// gets asked.
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
			if err != nil {
				t.Errorf("%s: %v", a.Slug, err)
				return
			}
			req.Header.Set("User-Agent", "behoerdenbarriere-seedcheck/0.1 (+https://github.com/praetorianer777/behoerdenbarriere)")

			resp, err := client.Do(req)
			if err != nil {
				// A host that does not resolve is a wrong URL. A connection that is cut
				// is usually a firewall that does not like scripted clients — Chrome
				// gets through where this request does not, so it is worth a note, not
				// a failure.
				var dnsErr *net.DNSError
				if errors.As(err, &dnsErr) {
					t.Errorf("%s (%s): host does not resolve: %v", a.Slug, a.URL, err)
					return
				}
				t.Logf("%s (%s): blocked or unreachable for a scripted client: %v", a.Slug, a.URL, err)
				return
			}
			defer resp.Body.Close()

			switch {
			case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
				t.Errorf("%s (%s): status %d — the URL is wrong", a.Slug, a.URL, resp.StatusCode)
			case resp.StatusCode >= 500:
				t.Logf("%s (%s): server error %d", a.Slug, a.URL, resp.StatusCode)
			case resp.StatusCode >= 400:
				// 400, 403 and 429 are what bot protection answers; several federal and
				// state portals turn away anything that is not a real browser.
				t.Logf("%s (%s): turned away with %d", a.Slug, a.URL, resp.StatusCode)
			}
		}()
	}
	wg.Wait()
}
