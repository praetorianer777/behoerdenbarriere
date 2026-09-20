package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

// A small landscape: two federal authorities with scores, one city without a scan.
func landscape(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()

	for _, a := range []struct {
		agency model.Agency
		scores []float64
		grades []string
	}{
		{model.Agency{Slug: "bmi", Name: "Bundesministerium des Innern", URL: "https://a.de/", Level: model.LevelBund},
			[]float64{60, 74.5}, []string{"D", "C"}},
		{model.Agency{Slug: "bmf", Name: "Bundesministerium der Finanzen", URL: "https://b.de/", Level: model.LevelBund},
			[]float64{91}, []string{"A"}},
		{model.Agency{Slug: "stadt-kiel", Name: "Kiel", URL: "https://c.de/", Level: model.LevelKommune, State: "Schleswig-Holstein"},
			nil, nil},
	} {
		id, err := s.UpsertAgency(ctx, a.agency)
		if err != nil {
			t.Fatalf("create authority: %v", err)
		}
		for i, score := range a.scores {
			age := len(a.scores) - i
			if _, err := s.Pool.Exec(ctx, `
				INSERT INTO scans (agency_id, status, started_at, finished_at, score, grade)
				VALUES ($1, 'done', now() - $2::interval, now() - $2::interval, $3, $4)`,
				id, fmt.Sprintf("%d days", age), score, a.grades[i]); err != nil {
				t.Fatalf("create scan: %v", err)
			}
		}
	}
}

func TestListAgenciesSortsByScoreWithUnscannedLast(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)

	got, total, err := s.ListAgencies(context.Background(), store.AgencyFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 || len(got) != 3 {
		t.Fatalf("%d of %d authorities", len(got), total)
	}
	if got[0].Slug != "bmf" || got[1].Slug != "bmi" {
		t.Fatalf("order = %s, %s", got[0].Slug, got[1].Slug)
	}
	// An authority nobody has checked is not the worst one — it is unknown, and it
	// belongs at the end either way.
	if got[2].Slug != "stadt-kiel" || got[2].Score != nil {
		t.Fatalf("unscanned authority: %+v", got[2])
	}

	ascending, _, err := s.ListAgencies(context.Background(), store.AgencyFilter{Sort: "score_asc"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if ascending[0].Slug != "bmi" || ascending[2].Slug != "stadt-kiel" {
		t.Fatalf("ascending order = %s ... %s", ascending[0].Slug, ascending[2].Slug)
	}
}

// The previous score comes along, so the ranking can show a change without a second
// query per row.
func TestListAgenciesCarriesThePreviousScore(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)

	got, _, err := s.ListAgencies(context.Background(), store.AgencyFilter{Query: "innern"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("%d hits", len(got))
	}
	if got[0].Score == nil || *got[0].Score != 74.5 {
		t.Fatalf("score = %v", got[0].Score)
	}
	if got[0].PrevScore == nil || *got[0].PrevScore != 60 {
		t.Fatalf("previous score = %v", got[0].PrevScore)
	}
}

func TestListAgenciesFilters(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)
	ctx := context.Background()

	cases := []struct {
		name   string
		filter store.AgencyFilter
		want   int
	}{
		{"by name", store.AgencyFilter{Query: "Kiel"}, 1},
		{"by slug", store.AgencyFilter{Query: "bmf"}, 1},
		{"by level", store.AgencyFilter{Level: "bund"}, 2},
		{"by state", store.AgencyFilter{State: "Schleswig-Holstein"}, 1},
		{"by grade", store.AgencyFilter{Grade: "A"}, 1},
		{"no match", store.AgencyFilter{Query: "Timbuktu"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, total, err := s.ListAgencies(ctx, c.filter)
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(got) != c.want || total != c.want {
				t.Fatalf("%d hits (total %d), want %d", len(got), total, c.want)
			}
		})
	}
}

func TestListAgenciesPaginates(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)
	ctx := context.Background()

	first, total, err := s.ListAgencies(ctx, store.AgencyFilter{PerPage: 2, Page: 1})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 || len(first) != 2 {
		t.Fatalf("page 1: %d of %d", len(first), total)
	}

	second, _, err := s.ListAgencies(ctx, store.AgencyFilter{PerPage: 2, Page: 2})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("page 2: %d", len(second))
	}
	if second[0].Slug == first[0].Slug {
		t.Fatal("the second page repeats the first")
	}
}

// A page size without a bound would let one request pull the whole table.
func TestListAgenciesCapsPageSize(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)

	got, _, err := s.ListAgencies(context.Background(), store.AgencyFilter{PerPage: 10000})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("%d authorities", len(got))
	}
}

func TestAgencyBySlug(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)
	ctx := context.Background()

	got, err := s.AgencyBySlug(ctx, "bmi")
	if err != nil {
		t.Fatalf("by slug: %v", err)
	}
	if got.Name == "" || got.Score == nil {
		t.Fatalf("authority = %+v", got)
	}
	if _, err := s.AgencyBySlug(ctx, "gibtsnicht"); err == nil {
		t.Fatal("an unknown slug was found")
	}
}

func TestStatsCountsAndAverages(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)

	got, err := s.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if got.Agencies != 3 {
		t.Errorf("agencies = %d", got.Agencies)
	}
	// The average is taken over the authorities that were actually checked; counting
	// the unchecked one as zero would invent a barrier nobody has seen.
	if got.Scanned != 2 || got.AvgScore == nil {
		t.Fatalf("scanned = %d, average = %v", got.Scanned, got.AvgScore)
	}
	if avg := *got.AvgScore; avg < 82.7 || avg > 82.8 {
		t.Errorf("average = %v, want about 82.75", avg)
	}
	if got.Grades["A"] != 1 || got.Grades["C"] != 1 {
		t.Errorf("grades = %v", got.Grades)
	}

	byLevel := map[string]int{}
	for _, g := range got.ByLevel {
		byLevel[g.Name] = g.Agencies
	}
	if byLevel["bund"] != 2 || byLevel["kommune"] != 1 {
		t.Errorf("by level = %v", byLevel)
	}
	if len(got.ByState) != 1 || got.ByState[0].Name != "Schleswig-Holstein" {
		t.Errorf("by state = %+v", got.ByState)
	}
}

func TestStatsOnAnEmptyDatabase(t *testing.T) {
	s := storetest.New(t)

	got, err := s.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if got.Agencies != 0 || got.AvgScore != nil {
		t.Fatalf("stats = %+v", got)
	}
}

func TestStatsCountsRulesPerAuthority(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	pages := []model.PageResult{
		{URL: "https://example.org/", IsEntry: true, DOMNodes: 800, Violations: []model.Violation{
			{RuleID: "color-contrast", Impact: model.ImpactSerious, Principle: model.Perceivable, NodeCount: 30},
		}},
		{URL: "https://example.org/a", DOMNodes: 800, Violations: []model.Violation{
			{RuleID: "color-contrast", Impact: model.ImpactSerious, Principle: model.Perceivable, NodeCount: 30},
		}},
	}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	got, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(got.TopRules) != 1 {
		t.Fatalf("rules = %+v", got.TopRules)
	}
	// One authority, two pages — an authority is counted once however many of its
	// pages are affected.
	if got.TopRules[0].Agencies != 1 || got.TopRules[0].Pages != 2 {
		t.Fatalf("rule count = %+v", got.TopRules[0])
	}
}

func TestStatesForTheFilter(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)

	got, err := s.States(context.Background())
	if err != nil {
		t.Fatalf("states: %v", err)
	}
	if len(got) != 1 || got[0] != "Schleswig-Holstein" {
		t.Fatalf("states = %v", got)
	}
}

func TestPagesForScanPutsTheEntryPageFirst(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	pages := []model.PageResult{
		{URL: "https://example.org/presse", DOMNodes: 800},
		{URL: "https://example.org/", IsEntry: true, DOMNodes: 800},
		{URL: "https://example.org/kontakt", Priority: true, DOMNodes: 800},
	}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	got, err := s.PagesForScan(ctx, scanID)
	if err != nil {
		t.Fatalf("pages: %v", err)
	}
	if len(got) != 3 || !got[0].IsEntry || !got[1].Priority {
		t.Fatalf("order = %+v", got)
	}
}

// The explanation of a score is rebuilt from what was stored, so the stored pages
// have to come back exactly as they went in — findings included.
func TestPageResultsForScanRebuildTheFindings(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	pages := []model.PageResult{
		{URL: "https://example.org/", Title: "Start", IsEntry: true, DOMNodes: 800,
			Violations: []model.Violation{
				{RuleID: "image-alt", Impact: model.ImpactCritical, Principle: model.Perceivable,
					Help: "Bilder brauchen eine Alternative", NodeCount: 12},
				{RuleID: "region", Impact: model.ImpactModerate, Principle: model.Robust, NodeCount: 1},
			}},
		{URL: "https://example.org/kontakt", Priority: true, DOMNodes: 600},
		{URL: "https://example.org/kaputt", Err: "timeout"},
	}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	got, err := s.PageResultsForScan(ctx, scanID)
	if err != nil {
		t.Fatalf("page results: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("%d pages", len(got))
	}

	byURL := map[string]model.PageResult{}
	for _, page := range got {
		byURL[page.URL] = page
	}
	entry := byURL["https://example.org/"]
	if !entry.IsEntry || entry.DOMNodes != 800 || len(entry.Violations) != 2 {
		t.Fatalf("entry page = %+v", entry)
	}
	if !byURL["https://example.org/kontakt"].Priority {
		t.Error("the priority page lost its mark")
	}
	if !byURL["https://example.org/kaputt"].Failed() {
		t.Error("the failed page lost its error")
	}

	// And the score computed from the rebuilt pages is the score that was stored.
	if rebuilt := scoring.SiteScore(got).Score; rebuilt != scoring.SiteScore(pages).Score {
		t.Fatalf("rebuilt score %v, stored %v", rebuilt, scoring.SiteScore(pages).Score)
	}
}

// bmi verbessert sich von 60 auf 74,5 (+14,5), bmf hat nur eine Prüfung und damit
// keine Veränderung, Kiel gar keine.
func TestListAgenciesSortsByChange(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)
	ctx := context.Background()

	descending, _, err := s.ListAgencies(ctx, store.AgencyFilter{Sort: "delta"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if descending[0].Slug != "bmi" {
		t.Errorf("biggest improvement = %s, want bmi", descending[0].Slug)
	}

	ascending, _, err := s.ListAgencies(ctx, store.AgencyFilter{Sort: "delta_asc"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if ascending[0].Slug != "bmi" {
		t.Errorf("only authority with a change = %s", ascending[0].Slug)
	}

	// Ohne Veränderung heißt nicht „Veränderung null": In beide Richtungen stehen
	// diese Behörden hinten, sonst führte eine unveränderte Liste die Rangfolge an.
	for _, order := range [][]store.AgencyListing{descending, ascending} {
		if order[0].PrevScore == nil {
			t.Fatalf("an authority without a change leads the order: %+v", order[0])
		}
	}
}

func TestListAgenciesSortsByLevelAndDate(t *testing.T) {
	s := storetest.New(t)
	landscape(t, s)
	ctx := context.Background()

	byLevel, _, err := s.ListAgencies(ctx, store.AgencyFilter{Sort: "level"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Nach Rang, nicht nach Alphabet: Bund vor Kommune, und innerhalb des Bundes nach
	// Namen.
	if byLevel[0].Level != model.LevelBund || byLevel[2].Level != model.LevelKommune {
		t.Errorf("order = %s, %s, %s", byLevel[0].Level, byLevel[1].Level, byLevel[2].Level)
	}
	if byLevel[0].Slug != "bmf" {
		t.Errorf("within the level it should go by name: %s", byLevel[0].Slug)
	}

	reversed, _, err := s.ListAgencies(ctx, store.AgencyFilter{Sort: "level_desc"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if reversed[0].Level != model.LevelKommune {
		t.Errorf("reversed order starts with %s", reversed[0].Level)
	}

	// Beide zuletzt geprüften Scans liegen in der Landschaft einen Tag zurück und
	// unterscheiden sich nur um Mikrosekunden. Ein Test, der darauf baut, prüft den
	// Zufall — also werden die Zeitpunkte hier auseinandergezogen.
	if _, err := s.Pool.Exec(ctx, `
		UPDATE scans SET finished_at = now() - interval '30 days'
		WHERE agency_id = (SELECT id FROM agencies WHERE slug = 'bmi')`); err != nil {
		t.Fatalf("age the scans: %v", err)
	}

	oldest, _, err := s.ListAgencies(ctx, store.AgencyFilter{Sort: "scanned_asc"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if oldest[0].Slug != "bmi" {
		t.Errorf("oldest check = %s, want bmi", oldest[0].Slug)
	}
	if oldest[2].Slug != "stadt-kiel" {
		t.Errorf("never checked should stay last, got %s", oldest[2].Slug)
	}

	newest, _, err := s.ListAgencies(ctx, store.AgencyFilter{Sort: "scanned"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if newest[0].Slug != "bmf" {
		t.Errorf("most recent check = %s, want bmf", newest[0].Slug)
	}
	// Auch bei der jüngsten Prüfung zuerst gehört „nie geprüft" ans Ende und nicht an
	// den Anfang.
	if newest[2].Slug != "stadt-kiel" {
		t.Errorf("never checked should stay last, got %s", newest[2].Slug)
	}
}
