package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
)

func TestAgencyCarriesItsMailRecord(t *testing.T) {
	db := &fakeDB{
		agencies: sampleAgencies(),
		mail: &maildns.Record{
			Domain:   "bmi.bund.de",
			Provider: maildns.Microsoft365,
			MX: []maildns.MXHost{{Host: "bmi-bund-de.mail.protection.outlook.com",
				Preference: 10, Provider: maildns.Microsoft365}},
			SPF:         "v=spf1 include:spf.protection.outlook.com -all",
			SPFIncludes: []string{"spf.protection.outlook.com"},
			DMARCPolicy: "reject",
		},
	}

	rec := request(t, db, http.MethodGet, "/api/v1/agencies/bmi", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := decode[agencyDetailDTO](t, rec)
	if got.Mail == nil {
		t.Fatal("no mail record")
	}
	if got.Mail.Provider != maildns.Microsoft365 {
		t.Errorf("provider = %q", got.Mail.Provider)
	}
	// Ohne den Rohwert ist die Einordnung nicht prüfbar.
	if len(got.Mail.MX) != 1 || got.Mail.MX[0].Host == "" {
		t.Errorf("mx = %+v", got.Mail.MX)
	}
}

// Not every authority has been looked up. A missing record is a gap in our data, not a
// statement about the authority, so the field is simply absent.
func TestAgencyWithoutAMailRecord(t *testing.T) {
	rec := request(t, &fakeDB{agencies: sampleAgencies()}, http.MethodGet,
		"/api/v1/agencies/bmi", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode[agencyDetailDTO](t, rec); got.Mail != nil {
		t.Errorf("mail = %+v, want none", got.Mail)
	}
}

func TestMailOverview(t *testing.T) {
	checked := time.Date(2026, 9, 20, 6, 0, 0, 0, time.UTC)
	db := &fakeDB{mailSummary: &store.MailSummary{
		Total:     3,
		CheckedAt: &checked,
		ByProvider: []store.MailCount{
			{Provider: "microsoft365", Agencies: 2},
			{Provider: "public-it", Agencies: 1},
		},
		ByState: []store.MailCount{{Name: "Bayern", Provider: "microsoft365", Agencies: 2}},
		ByLevel: []store.MailCount{{Name: "kommune", Provider: "microsoft365", Agencies: 2}},
	}}

	rec := request(t, db, http.MethodGet, "/api/v1/mail", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := decode[mailSummaryDTO](t, rec)
	if got.Total != 3 || got.CheckedAt == nil {
		t.Fatalf("summary = %+v", got)
	}
	if len(got.ByProvider) != 2 {
		t.Fatalf("by provider = %+v", got.ByProvider)
	}
	if !got.ByProvider[0].US {
		t.Error("Microsoft 365 should be marked as US-based")
	}
	// Ein öffentlicher IT-Dienstleister ist keine US-Cloud, und diese Zeile wird
	// zitiert werden.
	if got.ByProvider[1].US {
		t.Errorf("public-it must not be marked as US-based: %+v", got.ByProvider[1])
	}
}

func TestMailOverviewEmpty(t *testing.T) {
	rec := request(t, &fakeDB{}, http.MethodGet, "/api/v1/mail", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode[mailSummaryDTO](t, rec)
	if got.Total != 0 || len(got.ByProvider) != 0 {
		t.Errorf("summary = %+v", got)
	}
}
