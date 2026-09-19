package scanner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func TestResolveWebSocketURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"webSocketDebuggerUrl":"ws://localhost:9222/devtools/browser/abc"}`))
	}))
	defer srv.Close()

	got, err := resolveWebSocketURL(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("resolveWebSocketURL: %v", err)
	}

	// Chrome answers with the host it knows itself by, which inside a container is
	// "localhost" and from another container points at the wrong process. The session
	// path is what matters; the host has to stay the one we asked.
	want := "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/devtools/browser/abc"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

// A WebSocket URL configured by hand is used as is; asking Chrome about it would fail
// because the DevTools HTTP endpoint answers under a different scheme.
func TestResolveWebSocketURLPassesThroughWebSocketScheme(t *testing.T) {
	got, err := resolveWebSocketURL(context.Background(), "ws://chrome:9222/devtools/browser/xyz")
	if err != nil {
		t.Fatalf("resolveWebSocketURL: %v", err)
	}
	if got != "ws://chrome:9222/devtools/browser/xyz" {
		t.Fatalf("url = %q", got)
	}
}

func TestResolveWebSocketURLReportsUnreachableChrome(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	if _, err := resolveWebSocketURL(context.Background(), srv.URL); err == nil {
		t.Fatal("a response without webSocketDebuggerUrl was accepted")
	}
}

const brokenPage = `<!doctype html>
<html>
<head><title>Amt ohne Sorgfalt</title></head>
<body>
  <img src="wappen.png">
  <form><input type="text" name="suche"></form>
  <a href="/kontakt">Kontakt</a>
  <a href="https://example.org/extern">Extern</a>
</body>
</html>`

const cleanPage = `<!doctype html>
<html lang="de">
<head><title>Amt mit Sorgfalt</title></head>
<body>
  <main>
    <h1>Willkommen</h1>
    <img src="wappen.png" alt="Wappen der Stadt">
    <form>
      <label for="suche">Suche</label>
      <input type="text" id="suche" name="suche">
    </form>
    <a href="/kontakt">Kontakt</a>
  </main>
</body>
</html>`

// The scanner is only worth anything if it actually finds violations in a real browser.
// The test needs a Chrome: with CHROME_URL it uses a running one, otherwise a local
// binary, and without either it is skipped.
func newTestScanner(t *testing.T) *Scanner {
	t.Helper()
	if testing.Short() {
		t.Skip("browser test skipped in short mode")
	}
	opts := Options{ChromeURL: os.Getenv("CHROME_URL"), PageTimeout: 60 * time.Second}
	if opts.ChromeURL == "" && !localChromeAvailable() {
		t.Skip("neither CHROME_URL nor a local Chrome available")
	}

	s, err := New(context.Background(), opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func localChromeAvailable() bool {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "headless-shell"} {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}

func servePage(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestScanFindsKnownViolations(t *testing.T) {
	s := newTestScanner(t)
	got := s.Scan(context.Background(), servePage(t, brokenPage))

	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}
	if got.Result.HTTPStatus != http.StatusOK {
		t.Errorf("status = %d", got.Result.HTTPStatus)
	}
	if got.Result.Title != "Amt ohne Sorgfalt" {
		t.Errorf("title = %q", got.Result.Title)
	}
	if got.Result.DOMNodes < 5 {
		t.Errorf("DOM nodes = %d", got.Result.DOMNodes)
	}

	found := map[string]model.Violation{}
	for _, v := range got.Result.Violations {
		found[v.RuleID] = v
	}
	for _, rule := range []string{"image-alt", "html-has-lang"} {
		if _, ok := found[rule]; !ok {
			t.Errorf("rule %s not reported, got %v", rule, keys(found))
		}
	}
	if v, ok := found["image-alt"]; ok {
		if v.Impact != model.ImpactCritical {
			t.Errorf("image-alt impact = %s", v.Impact)
		}
		if v.Principle != model.Perceivable {
			t.Errorf("image-alt principle = %s", v.Principle)
		}
		if !strings.Contains(v.SampleHTML, "img") {
			t.Errorf("sample without the offending element: %q", v.SampleHTML)
		}
	}
}

func TestScanReturnsLinks(t *testing.T) {
	s := newTestScanner(t)
	base := servePage(t, brokenPage)
	got := s.Scan(context.Background(), base)

	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}
	var internal, external bool
	for _, link := range got.Links {
		if link == base+"/kontakt" {
			internal = true
		}
		if link == "https://example.org/extern" {
			external = true
		}
	}
	if !internal || !external {
		t.Fatalf("links incomplete: %v", got.Links)
	}
}

// Links are handed back absolute; a relative href would send the crawler nowhere.
func TestScanResolvesRelativeLinks(t *testing.T) {
	s := newTestScanner(t)
	got := s.Scan(context.Background(), servePage(t, brokenPage))
	for _, link := range got.Links {
		if !strings.HasPrefix(link, "http") {
			t.Fatalf("relative link: %q", link)
		}
	}
}

func TestScanCleanPageHasNoViolations(t *testing.T) {
	s := newTestScanner(t)
	got := s.Scan(context.Background(), servePage(t, cleanPage))

	if got.Result.Err != "" {
		t.Fatalf("scan failed: %s", got.Result.Err)
	}
	if len(got.Result.Violations) != 0 {
		t.Fatalf("clean page reports violations: %v", got.Result.Violations)
	}
}

// An unreachable page must not take the whole agency scan down with it.
func TestScanReportsFailureInResult(t *testing.T) {
	s := newTestScanner(t)
	got := s.Scan(context.Background(), "http://127.0.0.1:1/gibtesnicht")

	if got.Result.Err == "" {
		t.Fatal("no error recorded")
	}
	if !got.Result.Failed() {
		t.Fatal("Failed() = false")
	}
}

func keys(m map[string]model.Violation) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
