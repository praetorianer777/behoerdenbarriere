package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

func samplePages() []model.PageResult {
	return []model.PageResult{
		{
			URL: "https://example.org/", Title: "Start", IsEntry: true, DOMNodes: 900,
			HTTPStatus: 200, LoadMS: 1200,
			Violations: []model.Violation{{
				RuleID: "image-alt", Impact: model.ImpactCritical, Principle: model.Perceivable,
				WCAGTags: []string{"wcag2a", "wcag111"}, NodeCount: 4,
				SampleHTML: `<img src="x.png">`, SampleTarget: "img",
			}},
		},
		{URL: "https://example.org/kontakt", Priority: true, DOMNodes: 700, HTTPStatus: 200},
		{URL: "https://example.org/kaputt", Err: "timeout"},
	}
}

func TestFinishScanStoresEverything(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, map[string]any{"max_pages": 10})
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	pages := samplePages()
	result := scoring.SiteScore(pages)
	if err := s.FinishScan(ctx, scanID, pages, result); err != nil {
		t.Fatalf("finish: %v", err)
	}

	var status, grade string
	var score float64
	var scanned, failed int
	if err := s.Pool.QueryRow(ctx, `
		SELECT status, coalesce(grade, ''), coalesce(score, 0), pages_scanned, pages_failed
		FROM scans WHERE id = $1`, scanID,
	).Scan(&status, &grade, &score, &scanned, &failed); err != nil {
		t.Fatalf("read scan: %v", err)
	}
	if status != "done" {
		t.Errorf("status = %s", status)
	}
	if score != result.Score || grade != result.Grade {
		t.Errorf("score = %v/%s, want %v/%s", score, grade, result.Score, result.Grade)
	}
	// Two pages could be checked, one could not — and that difference has to survive,
	// because a scan of half a site says less than a scan of a whole one.
	if scanned != 2 || failed != 1 {
		t.Errorf("scanned = %d, failed = %d", scanned, failed)
	}

	var pageCount, violationCount int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM pages WHERE scan_id = $1`, scanID).Scan(&pageCount); err != nil {
		t.Fatalf("count pages: %v", err)
	}
	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM violations v JOIN pages p ON p.id = v.page_id WHERE p.scan_id = $1`,
		scanID).Scan(&violationCount); err != nil {
		t.Fatalf("count violations: %v", err)
	}
	if pageCount != 3 || violationCount != 1 {
		t.Fatalf("%d pages, %d violations stored", pageCount, violationCount)
	}

	var tags []string
	var nodeCount int
	var principle string
	if err := s.Pool.QueryRow(ctx, `
		SELECT wcag_tags, node_count, principle FROM violations v
		JOIN pages p ON p.id = v.page_id WHERE p.scan_id = $1`, scanID,
	).Scan(&tags, &nodeCount, &principle); err != nil {
		t.Fatalf("read violation: %v", err)
	}
	if len(tags) != 2 || nodeCount != 4 || principle != "perceivable" {
		t.Errorf("violation stored wrong: %v / %d / %s", tags, nodeCount, principle)
	}
}

// Once a scan is finished, the next one may start; otherwise an authority would be
// checked exactly once, ever.
func TestFinishScanReleasesTheAuthority(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := s.StartScan(ctx, agencyID, nil); !errors.Is(err, store.ErrScanInFlight) {
		t.Fatalf("second scan: err = %v, want store.ErrScanInFlight", err)
	}

	pages := samplePages()
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if _, err := s.StartScan(ctx, agencyID, nil); err != nil {
		t.Fatalf("no scan possible after the first: %v", err)
	}
}

func TestFailScanKeepsTheCause(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := s.FailScan(ctx, scanID, errors.New("robots.txt forbids the start page")); err != nil {
		t.Fatalf("fail: %v", err)
	}

	var status, cause string
	if err := s.Pool.QueryRow(ctx,
		`SELECT status, coalesce(error, '') FROM scans WHERE id = $1`, scanID).Scan(&status, &cause); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if status != "failed" || cause == "" {
		t.Fatalf("status = %s, cause = %q", status, cause)
	}
}

func TestLatestScore(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	// Without a scan there is no score, and that is not an error — it is an authority
	// nobody has looked at yet.
	score, grade, _, err := s.LatestScore(ctx, agencyID)
	if err != nil {
		t.Fatalf("latest score: %v", err)
	}
	if score != 0 || grade != "" {
		t.Fatalf("score = %v/%s without a scan", score, grade)
	}

	for _, s2 := range []struct {
		score float64
		grade string
		age   string
	}{{62, "D", "30 days"}, {81, "B", "1 day"}} {
		if _, err := s.Pool.Exec(ctx, `
			INSERT INTO scans (agency_id, status, started_at, finished_at, score, grade)
			VALUES ($1, 'done', now() - $2::interval, now() - $2::interval, $3, $4)`,
			agencyID, s2.age, s2.score, s2.grade); err != nil {
			t.Fatalf("create scan: %v", err)
		}
	}

	score, grade, at, err := s.LatestScore(ctx, agencyID)
	if err != nil {
		t.Fatalf("latest score: %v", err)
	}
	if score != 81 || grade != "B" || at.IsZero() {
		t.Fatalf("latest score = %v/%s at %v", score, grade, at)
	}
}
