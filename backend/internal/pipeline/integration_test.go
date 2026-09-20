package pipeline_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/crawler"
	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/pipeline"
	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

const startPage = `<!doctype html>
<html>
<head><title>Musteramt</title></head>
<body>
  <img src="wappen.png">
  <a href="/kontakt">Kontakt</a>
  <a href="/presse">Presse</a>
</body>
</html>`

const contactPage = `<!doctype html>
<html lang="de">
<head><title>Kontakt</title></head>
<body><main><h1>Kontakt</h1><form><input type="text" name="name"></form></main></body>
</html>`

const pressPage = `<!doctype html>
<html lang="de">
<head><title>Presse</title></head>
<body><main><h1>Presse</h1><p>Nichts Neues.</p></main></body>
</html>`

// Runs the whole chain once against a site of our own: crawl, check in a real browser,
// score, store. The parts are covered separately; what this test is for is that they
// fit together — that the scan ends up in the database with pages, violations and a
// grade, and that the authority is free for the next scan afterwards.
func TestRunAgainstARealSite(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}
	if os.Getenv("CHROME_URL") == "" && !localChrome() {
		t.Skip("neither CHROME_URL nor a local Chrome available")
	}

	db := storetest.New(t)
	ctx := context.Background()

	agencyID, err := db.UpsertAgency(ctx, model.Agency{
		Slug: "musteramt", Name: "Musteramt", URL: "https://example.org/", Level: model.LevelBund,
	})
	if err != nil {
		t.Fatalf("create authority: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", page(startPage))
	mux.HandleFunc("/kontakt", page(contactPage))
	mux.HandleFunc("/presse", page(pressPage))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	sc, err := scanner.New(ctx, scanner.Options{
		ChromeURL:   os.Getenv("CHROME_URL"),
		PageTimeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("scanner: %v", err)
	}
	defer sc.Close()

	cfg := crawler.Config{MaxPages: 5, MaxDepth: 2, RatePerSec: 50, Timeout: 3 * time.Minute}
	pipe := pipeline.New(db, crawler.New(sc, cfg), cfg, slog.Default())

	got, err := pipe.Run(ctx, model.Agency{ID: agencyID, Slug: "musteramt", URL: srv.URL})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Pages != 3 {
		t.Errorf("checked %d pages, want 3", got.Pages)
	}
	if got.Score.Grade == "" {
		t.Error("no grade")
	}

	var status, grade string
	var score float64
	var scanned int
	if err := db.Pool.QueryRow(ctx, `
		SELECT status, coalesce(grade, ''), coalesce(score, 0), pages_scanned
		FROM scans WHERE id = $1`, got.ScanID,
	).Scan(&status, &grade, &score, &scanned); err != nil {
		t.Fatalf("read scan: %v", err)
	}
	if status != "done" || scanned != 3 {
		t.Fatalf("scan: status = %s, pages = %d", status, scanned)
	}

	// The start page is the one with the violations, and the score has to show it.
	var entryScore, pressScore float64
	if err := db.Pool.QueryRow(ctx,
		`SELECT page_score FROM pages WHERE scan_id = $1 AND is_entry`, got.ScanID).Scan(&entryScore); err != nil {
		t.Fatalf("read entry page: %v", err)
	}
	if err := db.Pool.QueryRow(ctx,
		`SELECT page_score FROM pages WHERE scan_id = $1 AND url LIKE '%/presse'`, got.ScanID).Scan(&pressScore); err != nil {
		t.Fatalf("read press page: %v", err)
	}
	if entryScore >= pressScore {
		t.Errorf("the page with violations does not score worse: entry=%v press=%v", entryScore, pressScore)
	}

	var rules []string
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT v.rule_id FROM violations v
		JOIN pages p ON p.id = v.page_id WHERE p.scan_id = $1`, got.ScanID)
	if err != nil {
		t.Fatalf("read violations: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rule string
		if err := rows.Scan(&rule); err != nil {
			t.Fatalf("scan: %v", err)
		}
		rules = append(rules, rule)
	}
	if !contains(rules, "image-alt") || !contains(rules, "html-has-lang") {
		t.Errorf("expected violations missing: %v", rules)
	}

	// The contact page is reached as a priority page, so it has to be marked as one —
	// that is what weights it in the score.
	var priority bool
	if err := db.Pool.QueryRow(ctx,
		`SELECT priority FROM pages WHERE scan_id = $1 AND url LIKE '%/kontakt'`, got.ScanID).Scan(&priority); err != nil {
		t.Fatalf("read contact page: %v", err)
	}
	if !priority {
		t.Error("contact page not marked as a priority page")
	}

	// After a finished scan the authority is free again.
	if _, err := db.StartScan(ctx, agencyID, nil); err != nil {
		t.Fatalf("no second scan possible: %v", err)
	}
}

func page(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func localChrome() bool {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "headless-shell"} {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.EqualFold(s, needle) {
			return true
		}
	}
	return false
}
