package store_test

import (
	"context"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

// How far a scan reached past the consent layers has to survive into the database:
// a page checked behind a banner says something about the banner, and a reader can
// only weigh that if the number is there.
func TestFinishScanKeepsConsentState(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	pages := []model.PageResult{
		{URL: "https://example.org/", IsEntry: true, DOMNodes: 800, Consent: model.ConsentDeclined},
		{URL: "https://example.org/a", DOMNodes: 800, Consent: model.ConsentAccepted},
		{URL: "https://example.org/b", DOMNodes: 800, Consent: model.ConsentBlocked},
		{URL: "https://example.org/c", DOMNodes: 800},
	}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	detail, err := s.ScanByID(ctx, scanID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if detail.PagesBlocked != 1 {
		t.Errorf("blocked pages = %d, want 1", detail.PagesBlocked)
	}

	stored, err := s.PagesForScan(ctx, scanID)
	if err != nil {
		t.Fatalf("pages: %v", err)
	}
	states := map[string]string{}
	for _, p := range stored {
		states[p.URL] = p.Consent
	}
	want := map[string]string{
		"https://example.org/":  "declined",
		"https://example.org/a": "accepted",
		"https://example.org/b": "blocked",
		// A page that never met a layer must not look as if one had been dismissed.
		"https://example.org/c": "none",
	}
	for url, state := range want {
		if states[url] != state {
			t.Errorf("%s: consent = %q, want %q", url, states[url], state)
		}
	}
}

// The outside score is stored separately from the scan: the measurement may fail or
// arrive late, and neither may hold back our own result.
func TestSaveLighthouse(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	pages := []model.PageResult{{URL: "https://example.org/", IsEntry: true, DOMNodes: 800}}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	before, err := s.ScanByID(ctx, scanID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if before.LighthouseScore != nil {
		t.Fatalf("a score appeared without a measurement: %v", *before.LighthouseScore)
	}

	if err := s.SaveLighthouse(ctx, scanID, store.LighthouseResult{
		Score: 97, Failed: []string{"color-contrast"},
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	after, err := s.ScanByID(ctx, scanID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if after.LighthouseScore == nil || *after.LighthouseScore != 97 {
		t.Fatalf("score = %v", after.LighthouseScore)
	}
	if len(after.LighthouseFailed) != 1 || after.LighthouseFailed[0] != "color-contrast" {
		t.Fatalf("failed audits = %v", after.LighthouseFailed)
	}

	// The ranking carries it too, so both numbers can be compared at a glance.
	listing, _, err := s.ListAgencies(ctx, store.AgencyFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found bool
	for _, a := range listing {
		if a.ID == agencyID {
			found = a.LighthouseScore != nil && *a.LighthouseScore == 97
		}
	}
	if !found {
		t.Fatal("the ranking does not carry the outside score")
	}
}

// A grade computed mostly over consent banners has to be recognisable as such in the
// ranking too, not only on the authority's own page.
func TestRankingCarriesTheBlockedPageCount(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	pages := []model.PageResult{
		{URL: "https://example.org/", IsEntry: true, DOMNodes: 800, Consent: model.ConsentBlocked},
		{URL: "https://example.org/a", DOMNodes: 800, Consent: model.ConsentBlocked},
		{URL: "https://example.org/b", DOMNodes: 800, Consent: model.ConsentDeclined},
	}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	listing, _, err := s.ListAgencies(ctx, store.AgencyFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *store.AgencyListing
	for i, a := range listing {
		if a.ID == agencyID {
			found = &listing[i]
		}
	}
	if found == nil {
		t.Fatal("the authority is missing from the ranking")
	}
	if found.Pages != 3 || found.PagesBlocked != 2 {
		t.Fatalf("%d of %d pages blocked, want 2 of 3", found.PagesBlocked, found.Pages)
	}
	if !scoring.Obscured(found.PagesBlocked, found.Pages) {
		t.Error("a score over two banners out of three pages is not marked")
	}
}
