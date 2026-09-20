package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
)

func scanWithURL() *store.ScanDetail {
	scan := sampleScan()
	scan.AgencyURL = "https://www.bmi.bund.de/"
	return scan
}

func TestScanCarriesTheThirdParties(t *testing.T) {
	db := &fakeDB{
		agencies: sampleAgencies(),
		scanIDs:  []int64{99},
		scan:     scanWithURL(),
		seen: []thirdparty.Seen{
			{Host: "fonts.gstatic.com", Phase: model.PhaseBeforeConsent, Requests: 4, Pages: 2},
			{Host: "www.youtube.com", Phase: model.PhaseAfterAccepted, Requests: 1, Pages: 1},
			// The authority's own host is filtered out on the way out, wherever it
			// slipped in.
			{Host: "www.bmi.bund.de", Phase: model.PhaseBeforeConsent, Requests: 20, Pages: 2},
		},
	}

	rec := request(t, db, http.MethodGet, "/api/v1/scans/99", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := decode[scanDTO](t, rec)
	if len(got.ThirdParties) != 2 {
		t.Fatalf("third parties = %+v", got.ThirdParties)
	}
	// What went out before anyone could object comes first.
	first := got.ThirdParties[0]
	if first.Host != "fonts.gstatic.com" || first.Phase != model.PhaseBeforeConsent {
		t.Errorf("first = %+v", first)
	}
	if first.Group != thirdparty.GoogleFonts || first.Pages != 2 {
		t.Errorf("classification = %+v", first)
	}
}

// A scan from before we recorded contacts has none. Showing an empty list would claim
// the site contacts nobody, which we do not know.
func TestScanWithoutRecordedContactsShowsNoList(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies(), scanIDs: []int64{99}, scan: scanWithURL()}

	rec := request(t, db, http.MethodGet, "/api/v1/scans/99", nil)
	got := decode[scanDTO](t, rec)
	if got.ThirdParties != nil {
		t.Errorf("third parties = %+v, want none", got.ThirdParties)
	}
}

func TestThirdPartiesNationwide(t *testing.T) {
	db := &fakeDB{
		stats: &store.Stats{Scanned: 120},
		reach: []store.ContactReach{
			{Host: "fonts.gstatic.com", Phase: model.PhaseBeforeConsent, Agencies: 48, Pages: 900},
			{Host: "www.youtube.com", Phase: model.PhaseAfterAccepted, Agencies: 12, Pages: 30},
			{Host: "www.service.bund.de", Phase: model.PhaseBeforeConsent, Agencies: 9, Pages: 20},
		},
	}

	rec := request(t, db, http.MethodGet, "/api/v1/thirdparties", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := decode[thirdPartyListDTO](t, rec)
	// Without the denominator the counts mean nothing.
	if got.Scanned != 120 {
		t.Errorf("scanned = %d, want 120", got.Scanned)
	}
	if len(got.Items) != 3 {
		t.Fatalf("items = %+v", got.Items)
	}
	if got.Items[0].Group != string(thirdparty.GoogleFonts) || got.Items[0].Agencies != 48 {
		t.Errorf("first = %+v", got.Items[0])
	}
	if !got.Items[2].PublicBody {
		t.Errorf("a federal host should be marked as a public body: %+v", got.Items[2])
	}
	if got.Items[1].Phase != string(model.PhaseAfterAccepted) {
		t.Errorf("phase = %q", got.Items[1].Phase)
	}
}

func TestThirdPartiesEmptyIsAnEmptyArray(t *testing.T) {
	rec := request(t, &fakeDB{stats: &store.Stats{}}, http.MethodGet, "/api/v1/thirdparties", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"items":[]`) {
		t.Errorf("body = %s", body)
	}
}
