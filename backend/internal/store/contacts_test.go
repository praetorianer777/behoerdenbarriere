package store_test

import (
	"context"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
)

func TestContactsSurviveAScan(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	scanID, err := s.StartScan(ctx, agencyID, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	pages := []model.PageResult{
		{URL: "https://example.org/", IsEntry: true, DOMNodes: 800, Contacts: []model.Contact{
			{Host: "fonts.gstatic.com", Phase: model.PhaseBeforeConsent, Requests: 3},
			{Host: "www.youtube.com", Phase: model.PhaseAfterAccepted, Requests: 1},
		}},
		{URL: "https://example.org/kontakt", DOMNodes: 800, Contacts: []model.Contact{
			{Host: "fonts.gstatic.com", Phase: model.PhaseBeforeConsent, Requests: 2},
		}},
	}
	if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
		t.Fatalf("finish: %v", err)
	}

	seen, err := s.ContactsForScan(ctx, scanID)
	if err != nil {
		t.Fatalf("contacts: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("rows = %d, want 2: %+v", len(seen), seen)
	}

	fonts := seen[0]
	if fonts.Host != "fonts.gstatic.com" || fonts.Phase != model.PhaseBeforeConsent {
		t.Fatalf("first row = %+v", fonts)
	}
	if fonts.Pages != 2 || fonts.Requests != 5 {
		t.Errorf("fonts: pages = %d, requests = %d, want 2 and 5", fonts.Pages, fonts.Requests)
	}

	// The classification happens on the way out, not in the database.
	described := thirdparty.Describe(seen, "example.org")
	if len(described) != 2 || described[0].Group != thirdparty.GoogleFonts {
		t.Errorf("described = %+v", described)
	}
}

// A host the authority has since removed must not keep counting against it: only the
// newest finished scan of each authority feeds the nationwide picture.
func TestContactReachUsesTheNewestScanOnly(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	finish := func(host string) {
		t.Helper()
		scanID, err := s.StartScan(ctx, agencyID, nil)
		if err != nil {
			t.Fatalf("start: %v", err)
		}
		pages := []model.PageResult{{URL: "https://example.org/", IsEntry: true, DOMNodes: 800,
			Contacts: []model.Contact{{Host: host, Phase: model.PhaseBeforeConsent, Requests: 1}}}}
		if err := s.FinishScan(ctx, scanID, pages, scoring.SiteScore(pages)); err != nil {
			t.Fatalf("finish: %v", err)
		}
	}
	finish("www.google-analytics.com")
	finish("fonts.gstatic.com")

	reach, err := s.ContactReach(ctx, 100)
	if err != nil {
		t.Fatalf("reach: %v", err)
	}

	hosts := map[string]int{}
	for _, r := range reach {
		hosts[r.Host] = r.Agencies
	}
	if hosts["fonts.gstatic.com"] != 1 {
		t.Errorf("the host of the newest scan is missing: %+v", reach)
	}
	if _, ok := hosts["www.google-analytics.com"]; ok {
		t.Errorf("a host from an older scan is still counted: %+v", reach)
	}
}
