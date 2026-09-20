package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

func record(provider maildns.Provider, host string) maildns.Record {
	return maildns.Record{
		Domain:      "example.org",
		Provider:    provider,
		MX:          []maildns.MXHost{{Host: host, Preference: 10, Provider: provider}},
		SPF:         "v=spf1 include:spf.protection.outlook.com -all",
		SPFIncludes: []string{"spf.protection.outlook.com"},
		DMARC:       "v=DMARC1; p=reject",
		DMARCPolicy: "reject",
		CheckedAt:   time.Now().UTC().Truncate(time.Second),
	}
}

// The raw record has to come back as it went in: our classification may change, the
// published record is the evidence.
func TestMailRecordSurvivesTheRoundTrip(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	agencyID := freshAgency(t, s)

	want := record(maildns.Microsoft365, "example-org.mail.protection.outlook.com")
	if err := s.SaveMail(ctx, agencyID, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.MailForAgency(ctx, agencyID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Provider != want.Provider || got.DMARCPolicy != "reject" {
		t.Errorf("record = %+v", got)
	}
	if len(got.MX) != 1 || got.MX[0].Host != want.MX[0].Host {
		t.Errorf("mx = %+v", got.MX)
	}
	if len(got.SPFIncludes) != 1 {
		t.Errorf("spf includes = %v", got.SPFIncludes)
	}

	// A second look replaces the first: this is the current state of a domain, not a
	// history of it.
	later := record(maildns.PublicIT, "mx1.dataport.de")
	if err := s.SaveMail(ctx, agencyID, later); err != nil {
		t.Fatalf("save again: %v", err)
	}
	got, err = s.MailForAgency(ctx, agencyID)
	if err != nil {
		t.Fatalf("read again: %v", err)
	}
	if got.Provider != maildns.PublicIT {
		t.Errorf("provider = %q after the second lookup", got.Provider)
	}
}

func TestMailForAgencyWithoutARecord(t *testing.T) {
	s := storetest.New(t)
	if _, err := s.MailForAgency(context.Background(), freshAgency(t, s)); err != store.ErrNotFound {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestMailOverviewCountsByProviderStateAndLevel(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	first := freshAgency(t, s)
	second := freshAgency(t, s)
	if err := s.SaveMail(ctx, first, record(maildns.Microsoft365, "a.outlook.com")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SaveMail(ctx, second, record(maildns.Microsoft365, "b.outlook.com")); err != nil {
		t.Fatalf("save: %v", err)
	}

	summary, err := s.MailOverview(ctx)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if summary.Total != 2 {
		t.Errorf("total = %d, want 2", summary.Total)
	}
	if len(summary.ByProvider) != 1 || summary.ByProvider[0].Agencies != 2 {
		t.Errorf("by provider = %+v", summary.ByProvider)
	}
	if summary.CheckedAt == nil {
		t.Error("without a date nobody knows how old the picture is")
	}
	// freshAgency creates federal authorities, so both land in one level.
	if len(summary.ByLevel) != 1 || summary.ByLevel[0].Name != "bund" {
		t.Errorf("by level = %+v", summary.ByLevel)
	}
}
