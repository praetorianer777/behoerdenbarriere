package maildns_test

import (
	"context"
	"net"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
)

type fakeResolver struct {
	mx  map[string][]*net.MX
	txt map[string][]string
	err error
}

func (f fakeResolver) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	if f.err != nil {
		return nil, f.err
	}
	hosts, ok := f.mx[name]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
	return hosts, nil
}

func (f fakeResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	records, ok := f.txt[name]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
	return records, nil
}

func TestClassify(t *testing.T) {
	cases := map[string]maildns.Provider{
		"musterstadt-de.mail.protection.outlook.com": maildns.Microsoft365,
		"aspmx.l.google.com":                         maildns.GoogleWorkspace,
		"mx1.dataport.de":                            maildns.PublicIT,
		"mx00.kundenserver.de":                       maildns.IONOS,
		"mx-ha01.web.de":                             maildns.UnknownProvider,
		"mxlb.ispgateway.de":                         maildns.UnknownProvider,
		"mx2.mailbox.org":                            maildns.MailboxOrg,
		"mx1.itzbund.de":                             maildns.PublicIT,
	}
	for host, want := range cases {
		if got := maildns.Classify(host); got != want {
			t.Errorf("%s: provider = %q, want %q", host, got, want)
		}
	}
}

func TestLookupReadsMXAndTakesTheMostPreferred(t *testing.T) {
	client := maildns.New(fakeResolver{
		mx: map[string][]*net.MX{"musterstadt.de": {
			{Host: "backup.musterstadt.de.", Pref: 30},
			{Host: "musterstadt-de.mail.protection.outlook.com.", Pref: 10},
		}},
		txt: map[string][]string{
			"musterstadt.de": {
				"google-site-verification=abc",
				"v=spf1 include:spf.protection.outlook.com include:newsletter.example.net -all",
			},
			"_dmarc.musterstadt.de": {"v=DMARC1; p=quarantine; rua=mailto:dmarc@musterstadt.de"},
		},
	}, 0)

	got := client.Lookup(context.Background(), "Musterstadt.de.")

	if got.Domain != "musterstadt.de" {
		t.Errorf("domain = %q", got.Domain)
	}
	// The record with the lowest preference is the one that receives the mail.
	if len(got.MX) != 2 || got.MX[0].Host != "musterstadt-de.mail.protection.outlook.com" {
		t.Fatalf("mx = %+v", got.MX)
	}
	if got.Provider != maildns.Microsoft365 {
		t.Errorf("provider = %q", got.Provider)
	}
	if got.SPF == "" || len(got.SPFIncludes) != 2 {
		t.Errorf("spf = %q, includes = %v", got.SPF, got.SPFIncludes)
	}
	if got.DMARCPolicy != "quarantine" {
		t.Errorf("dmarc policy = %q", got.DMARCPolicy)
	}
	if got.Err != "" {
		t.Errorf("error = %q", got.Err)
	}
}

// An SPF include says a service may send in the domain's name. It says nothing about
// where the mail is kept — treating it as mail hosting is the mistake this kind of
// analysis usually makes, so the two are recorded in different fields.
func TestSPFSendersAreNotTheMailProvider(t *testing.T) {
	client := maildns.New(fakeResolver{
		mx: map[string][]*net.MX{"musterstadt.de": {{Host: "mx1.dataport.de.", Pref: 10}}},
		txt: map[string][]string{
			"musterstadt.de": {"v=spf1 include:_spf.google.com include:spf.protection.outlook.com ~all"},
		},
	}, 0)

	got := client.Lookup(context.Background(), "musterstadt.de")

	if got.Provider != maildns.PublicIT {
		t.Fatalf("provider = %q, want the operator of the MX", got.Provider)
	}
	if len(got.SPFSenders) != 2 {
		t.Fatalf("senders = %v", got.SPFSenders)
	}
}

// A domain that takes no mail is a finding. A lookup that failed is a gap. They must
// not look the same.
func TestNoMailIsNotAFailedLookup(t *testing.T) {
	none := maildns.New(fakeResolver{}, 0).Lookup(context.Background(), "nichts.de")
	if none.Provider != maildns.NoMail || none.Err != "" {
		t.Errorf("without MX: provider = %q, error = %q", none.Provider, none.Err)
	}

	broken := maildns.New(fakeResolver{err: &net.DNSError{Err: "server misbehaving", IsTemporary: true}}, 0).
		Lookup(context.Background(), "kaputt.de")
	if broken.Err == "" {
		t.Error("a failed lookup has to say so")
	}
	if broken.Provider == maildns.NoMail {
		t.Error("a failed lookup must not read as 'takes no mail'")
	}
}

// The null MX of RFC 7505 is a single dot: the domain states it accepts no mail.
func TestNullMX(t *testing.T) {
	got := maildns.New(fakeResolver{
		mx: map[string][]*net.MX{"nix.de": {{Host: ".", Pref: 0}}},
	}, 0).Lookup(context.Background(), "nix.de")

	if got.Provider != maildns.NoMail || len(got.MX) != 0 {
		t.Errorf("null MX: provider = %q, mx = %+v", got.Provider, got.MX)
	}
}

func TestSelfOperated(t *testing.T) {
	client := func(hosts ...*net.MX) maildns.Record {
		return maildns.New(fakeResolver{mx: map[string][]*net.MX{"musterstadt.de": hosts}}, 0).
			Lookup(context.Background(), "musterstadt.de")
	}

	got := client(&net.MX{Host: "mail.musterstadt.de.", Pref: 10},
		&net.MX{Host: "mail2.musterstadt.de.", Pref: 20})
	if got.Provider != maildns.SelfOperated {
		t.Errorf("provider = %q, want self", got.Provider)
	}

	// One exchanger outside the domain is enough: the mail leaves the house.
	got = client(&net.MX{Host: "mail.musterstadt.de.", Pref: 10},
		&net.MX{Host: "mx.anbieter.de.", Pref: 20})
	if got.Provider == maildns.SelfOperated {
		t.Errorf("provider = %q: an outside exchanger must not read as self-operated", got.Provider)
	}

	// A known operator is never overwritten, however the host is named.
	got = client(&net.MX{Host: "musterstadt-de.mail.protection.outlook.com.", Pref: 10})
	if got.Provider != maildns.Microsoft365 {
		t.Errorf("provider = %q, want microsoft365", got.Provider)
	}
}

// A ministry receives its mail at mx1.bund.de, a state authority at a host of its
// state. No suffix list of providers knows those, and "unknown" would hide exactly the
// case this data is collected for: administration that runs its own mail.
func TestPublicSectorHostsAreNotUnknown(t *testing.T) {
	got := maildns.New(fakeResolver{mx: map[string][]*net.MX{
		"bmi.bund.de": {{Host: "mx1.bund.de.", Pref: 10}, {Host: "mx2.bund.de.", Pref: 10}},
	}}, 0).Lookup(context.Background(), "bmi.bund.de")

	if got.Provider != maildns.PublicIT {
		t.Errorf("provider = %q, want public-it", got.Provider)
	}
}

func TestUSBasedCoversTheObviousOnes(t *testing.T) {
	for _, provider := range []maildns.Provider{maildns.Microsoft365, maildns.GoogleWorkspace} {
		if !maildns.USBased[provider] {
			t.Errorf("%q should be marked as US-based", provider)
		}
	}
	for _, provider := range []maildns.Provider{maildns.PublicIT, maildns.SelfOperated,
		maildns.Telekom, maildns.UnknownProvider} {
		if maildns.USBased[provider] {
			t.Errorf("%q must not be marked as US-based", provider)
		}
	}
}
