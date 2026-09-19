package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

func TestScanHistoryIsOldestFirst(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	for _, sc := range []struct {
		age   string
		score float64
		grade string
	}{{"90 days", 55, "E"}, {"30 days", 68, "D"}, {"1 day", 74, "C"}} {
		if _, err := s.Pool.Exec(ctx, `
			INSERT INTO scans (agency_id, status, started_at, finished_at, score, grade)
			VALUES ($1, 'done', now() - $2::interval, now() - $2::interval, $3, $4)`,
			agencyID, sc.age, sc.score, sc.grade); err != nil {
			t.Fatalf("create scan: %v", err)
		}
	}
	// A running scan has no score yet and must not show up as a dip in the curve.
	if _, err := s.Pool.Exec(ctx,
		`INSERT INTO scans (agency_id, status) VALUES ($1, 'running')`, agencyID); err != nil {
		t.Fatalf("create running scan: %v", err)
	}

	got, err := s.ScanHistory(ctx, agencyID, 50)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("%d points, want 3", len(got))
	}
	if got[0].Score != 55 || got[2].Score != 74 {
		t.Fatalf("order wrong: %v", got)
	}
	for i := 1; i < len(got); i++ {
		if !got[i].At.After(got[i-1].At) {
			t.Fatalf("not sorted by time: %v", got)
		}
	}
}

// The limit has to keep the newest scans; a curve of the oldest ones would show the
// past and call it the present.
func TestScanHistoryLimitKeepsTheNewest(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	for i := range 5 {
		if _, err := s.Pool.Exec(ctx, `
			INSERT INTO scans (agency_id, status, started_at, finished_at, score, grade)
			VALUES ($1, 'done', now() - $2::interval, now() - $2::interval, $3, 'C')`,
			agencyID, time.Duration(i*24)*time.Hour, float64(70+i)); err != nil {
			t.Fatalf("create scan: %v", err)
		}
	}

	got, err := s.ScanHistory(ctx, agencyID, 2)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(got) != 2 || got[1].Score != 70 {
		t.Fatalf("history = %v", got)
	}
}

func TestRulesForScanAggregatesAcrossPages(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	contrast := model.Violation{
		RuleID: "color-contrast", Impact: model.ImpactSerious, Principle: model.Perceivable,
		Help: "Contrast too low", HelpURL: "https://example.org/contrast",
		SampleHTML: "<a>x</a>", SampleTarget: "a",
	}
	pages := []model.PageResult{
		{URL: "https://example.org/", IsEntry: true, DOMNodes: 800, Violations: []model.Violation{
			withNodes(contrast, 10),
			{RuleID: "label", Impact: model.ImpactCritical, Principle: model.Perceivable, NodeCount: 1},
		}},
		{URL: "https://example.org/a", DOMNodes: 800, Violations: []model.Violation{withNodes(contrast, 5)}},
	}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	got, err := s.RulesForScan(ctx, scanID)
	if err != nil {
		t.Fatalf("rules: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("%d rules, want 2", len(got))
	}
	// The severe rule comes first, whichever page it was found on.
	if got[0].RuleID != "label" {
		t.Errorf("order = %s first", got[0].RuleID)
	}

	var contrastRule *scoring.RuleSummary
	for i := range got {
		if got[i].RuleID == "color-contrast" {
			contrastRule = &got[i]
		}
	}
	if contrastRule == nil {
		t.Fatal("color-contrast missing")
	}
	if contrastRule.Pages != 2 || contrastRule.Nodes != 15 {
		t.Errorf("aggregation: %d pages, %d elements", contrastRule.Pages, contrastRule.Nodes)
	}
	if contrastRule.Sample == nil || contrastRule.Sample.SampleTarget != "a" {
		t.Errorf("sample missing: %+v", contrastRule.Sample)
	}
	if contrastRule.HelpURL == "" {
		t.Error("help URL lost")
	}
}

func TestLastScanIDsAreNewestFirst(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	var wantNewest int64
	for i := range 3 {
		var id int64
		if err := s.Pool.QueryRow(ctx, `
			INSERT INTO scans (agency_id, status, started_at, finished_at, score, grade)
			VALUES ($1, 'done', now() - $2::interval, now() - $2::interval, 70, 'C')
			RETURNING id`, agencyID, time.Duration(i*24)*time.Hour).Scan(&id); err != nil {
			t.Fatalf("create scan: %v", err)
		}
		if i == 0 {
			wantNewest = id
		}
	}

	got, err := s.LastScanIDs(ctx, agencyID, 2)
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	if len(got) != 2 || got[0] != wantNewest {
		t.Fatalf("ids = %v, newest = %d", got, wantNewest)
	}
}

func withNodes(v model.Violation, nodes int) model.Violation {
	v.NodeCount = nodes
	return v
}
