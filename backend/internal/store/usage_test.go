package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
	"github.com/praetorianer777/behoerdenbarriere/internal/usage"
)

func today(t *testing.T, s *store.Store) time.Time {
	t.Helper()
	var day time.Time
	if err := s.Pool.QueryRow(context.Background(), `SELECT current_date`).Scan(&day); err != nil {
		t.Fatalf("current date: %v", err)
	}
	return day
}

func TestAddUsageSumsInsteadOfPilingUpRows(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	day := today(t, s)

	batch := []usage.Count{
		{Day: day, Kind: usage.KindPage, Key: "/", Count: 3},
		{Day: day, Kind: usage.KindAPI, Key: "GET /api/v1/agencies", Count: 3},
	}
	for range 2 {
		if err := s.AddUsage(ctx, batch); err != nil {
			t.Fatalf("add usage: %v", err)
		}
	}

	var rows int
	var views int64
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*), coalesce(sum(count) FILTER (WHERE kind = 'page'), 0) FROM usage_counters`,
	).Scan(&rows, &views); err != nil {
		t.Fatalf("read counters: %v", err)
	}
	if rows != 2 {
		t.Errorf("%d rows after two batches, want 2", rows)
	}
	if views != 6 {
		t.Errorf("page views = %d, want 6", views)
	}
}

func TestUsageSummarizesTheWindow(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	day := today(t, s)
	yesterday := day.AddDate(0, 0, -1)
	longAgo := day.AddDate(0, 0, -40)

	if err := s.AddUsage(ctx, []usage.Count{
		{Day: day, Kind: usage.KindVisitor, Key: "hash-a", Count: 4},
		{Day: day, Kind: usage.KindVisitor, Key: "hash-b", Count: 1},
		{Day: day, Kind: usage.KindPage, Key: "/", Count: 3},
		{Day: day, Kind: usage.KindPage, Key: "/behoerde/:slug", Count: 2},
		{Day: day, Kind: usage.KindAgency, Key: "stadt-kiel", Count: 2},
		{Day: day, Kind: usage.KindAPI, Key: "GET /api/v1/agencies", Count: 7},
		{Day: yesterday, Kind: usage.KindVisitor, Key: "hash-c", Count: 2},
		{Day: yesterday, Kind: usage.KindPage, Key: "/", Count: 2},
		{Day: longAgo, Kind: usage.KindVisitor, Key: "hash-d", Count: 9},
		{Day: longAgo, Kind: usage.KindPage, Key: "/", Count: 9},
	}); err != nil {
		t.Fatalf("add usage: %v", err)
	}

	summary, err := s.Usage(ctx, 30)
	if err != nil {
		t.Fatalf("usage: %v", err)
	}

	if summary.Visitors != 3 {
		t.Errorf("visitors = %d, want 3 — counted per hash, not per request", summary.Visitors)
	}
	if summary.Views != 7 {
		t.Errorf("page views = %d, want 7", summary.Views)
	}
	if len(summary.Days) != 2 {
		t.Fatalf("%d days in the window, want 2 — the 40 day old one is outside", len(summary.Days))
	}
	if !summary.Days[0].Day.Equal(yesterday) {
		t.Errorf("days start at %v, want %v", summary.Days[0].Day, yesterday)
	}
	if len(summary.Pages) != 2 || summary.Pages[0].Key != "/" || summary.Pages[0].Count != 5 {
		t.Errorf("pages = %+v, want the home page in front with 5", summary.Pages)
	}
	if len(summary.Agencies) != 1 || summary.Agencies[0].Key != "stadt-kiel" {
		t.Errorf("authorities = %+v", summary.Agencies)
	}
	if len(summary.Endpoints) != 1 || summary.Endpoints[0].Count != 7 {
		t.Errorf("endpoints = %+v", summary.Endpoints)
	}
	for _, page := range summary.Pages {
		if page.Key == "hash-a" {
			t.Fatal("visitor hashes are handed out as pages")
		}
	}
}

func TestDropVisitorHashesLeavesTheCountersAlone(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	day := today(t, s)
	old := day.AddDate(0, 0, -31)

	if err := s.AddUsage(ctx, []usage.Count{
		{Day: old, Kind: usage.KindVisitor, Key: "hash-old", Count: 1},
		{Day: old, Kind: usage.KindPage, Key: "/", Count: 5},
		{Day: day, Kind: usage.KindVisitor, Key: "hash-new", Count: 1},
	}); err != nil {
		t.Fatalf("add usage: %v", err)
	}

	if err := s.DropVisitorHashes(ctx, day.AddDate(0, 0, -30)); err != nil {
		t.Fatalf("drop: %v", err)
	}

	var hashes, pages int
	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE kind = 'visitor'), count(*) FILTER (WHERE kind = 'page')
		FROM usage_counters`).Scan(&hashes, &pages); err != nil {
		t.Fatalf("count: %v", err)
	}
	if hashes != 1 {
		t.Errorf("%d visitor rows left, want only the recent one", hashes)
	}
	if pages != 1 {
		t.Errorf("%d page rows left, want the old page counter kept", pages)
	}
}
