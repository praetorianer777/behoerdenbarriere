package thirdparty_test

import (
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
)

func TestClassify(t *testing.T) {
	cases := map[string]thirdparty.Group{
		"fonts.googleapis.com":     thirdparty.GoogleFonts,
		"fonts.gstatic.com":        thirdparty.GoogleFonts,
		"www.google-analytics.com": thirdparty.GoogleAnalytics,
		"www.googletagmanager.com": thirdparty.GoogleAnalytics,
		"www.youtube-nocookie.com": thirdparty.YouTube,
		"i.ytimg.com":              thirdparty.YouTube,
		"player.vimeo.com":         thirdparty.Vimeo,
		"connect.facebook.net":     thirdparty.Meta,
		"platform.twitter.com":     thirdparty.X,
		"snap.licdn.com":           thirdparty.LinkedIn,
		"matomo.stadt-musterau.de": thirdparty.Matomo,
		"piwik.example.org":        thirdparty.Matomo,
		"cdn.jsdelivr.net":         thirdparty.CDN,
		"consent.cookiebot.com":    thirdparty.ConsentTool,
		"maps.googleapis.com":      thirdparty.GoogleMaps,
		"www.gstatic.com":          thirdparty.GoogleOther,
		"www.bundesregierung.de":   thirdparty.Unknown,
		"static.stadt-musterau.de": thirdparty.Unknown,
	}
	for host, want := range cases {
		if got := thirdparty.Classify(host); got != want {
			t.Errorf("%s: group = %q, want %q", host, got, want)
		}
	}
}

// YouTube runs on Google's infrastructure, and a rule for Google's domains placed
// first would swallow it. An embedded video and a font are different decisions and
// have to stay apart.
func TestYouTubeIsNotFiledUnderGoogle(t *testing.T) {
	for _, host := range []string{"www.youtube.com", "i.ytimg.com", "r1---sn-x.googlevideo.com"} {
		if got := thirdparty.Classify(host); got != thirdparty.YouTube {
			t.Errorf("%s: group = %q, want youtube", host, got)
		}
	}
}

func TestPublicBody(t *testing.T) {
	yes := []string{"www.bund.de", "service.bund.de", "www.berlin.de", "id.land.nrw", "europa.eu"}
	no := []string{"fonts.gstatic.com", "www.youtube.com", "stadt-musterau.de"}
	for _, host := range yes {
		if !thirdparty.PublicBody(host) {
			t.Errorf("%s should count as a public body", host)
		}
	}
	for _, host := range no {
		if thirdparty.PublicBody(host) {
			t.Errorf("%s should not count as a public body", host)
		}
	}
}

func TestRegistrableAndSameSite(t *testing.T) {
	// bund.de is a registrable domain, so every ministry shares it. For this purpose
	// that is the right answer: a federal host is not an outsider to another federal
	// host, however much the crawler has to tell them apart.
	if got := thirdparty.Registrable("www.bmi.bund.de"); got != "bund.de" {
		t.Errorf("registrable = %q, want bund.de", got)
	}
	if got := thirdparty.Registrable("fonts.gstatic.com:443"); got != "gstatic.com" {
		t.Errorf("registrable with port = %q", got)
	}
	if !thirdparty.SameSite("www.bmi.bund.de", "bmi.bund.de") {
		t.Error("www and bare host belong to the same site")
	}
	// bund.de is the registrable domain of every federal ministry, so two ministries
	// would look like one site. That is the crawler's problem; here it only means a
	// federal host is not treated as an outsider, which is what we want.
	if !thirdparty.SameSite("www.bmi.bund.de", "www.bmf.bund.de") {
		t.Error("both are under bund.de")
	}
	if thirdparty.SameSite("www.bmi.bund.de", "fonts.gstatic.com") {
		t.Error("gstatic.com is not bund.de")
	}
}

func TestSummarizeFoldsPagesAndKeepsPhases(t *testing.T) {
	pages := []model.PageResult{
		{URL: "https://stadt-musterau.de/", Contacts: []model.Contact{
			{Host: "fonts.gstatic.com", Phase: model.PhaseBeforeConsent, Requests: 3},
			{Host: "www.youtube.com", Phase: model.PhaseAfterAccepted, Requests: 1},
			{Host: "www.stadt-musterau.de", Phase: model.PhaseBeforeConsent, Requests: 9},
		}},
		{URL: "https://stadt-musterau.de/kontakt", Contacts: []model.Contact{
			{Host: "fonts.gstatic.com", Phase: model.PhaseBeforeConsent, Requests: 2},
			{Host: "fonts.gstatic.com", Phase: model.PhaseAfterDeclined, Requests: 1},
		}},
	}

	got := thirdparty.Summarize(pages, "stadt-musterau.de")

	// The site's own host is not a third party and must not appear.
	for _, obs := range got {
		if obs.Host == "www.stadt-musterau.de" {
			t.Fatal("the site's own host was recorded as a third party")
		}
	}
	if len(got) != 3 {
		t.Fatalf("observations = %d, want 3: %+v", len(got), got)
	}

	first := got[0]
	if first.Host != "fonts.gstatic.com" || first.Phase != model.PhaseBeforeConsent {
		t.Fatalf("first observation = %+v", first)
	}
	if first.Pages != 2 || first.Requests != 5 {
		t.Errorf("fonts before consent: pages = %d, requests = %d, want 2 and 5",
			first.Pages, first.Requests)
	}
	if first.Group != thirdparty.GoogleFonts || first.Domain != "gstatic.com" {
		t.Errorf("classification = %q / %q", first.Group, first.Domain)
	}

	// The same host in two phases stays two findings: they say different things.
	var declined bool
	for _, obs := range got {
		if obs.Host == "fonts.gstatic.com" && obs.Phase == model.PhaseAfterDeclined {
			declined = true
		}
	}
	if !declined {
		t.Error("the contact after declining was folded into the one before")
	}
}

// A page that never met a consent layer has no later phase — everything it did, it did
// without being asked.
func TestBeforeConsentLeavesOutPublicBodies(t *testing.T) {
	pages := []model.PageResult{{Contacts: []model.Contact{
		{Host: "fonts.gstatic.com", Phase: model.PhaseBeforeConsent, Requests: 1},
		{Host: "www.service.bund.de", Phase: model.PhaseBeforeConsent, Requests: 1},
		{Host: "www.youtube.com", Phase: model.PhaseAfterAccepted, Requests: 1},
	}}}

	got := thirdparty.BeforeConsent(thirdparty.Summarize(pages, "stadt-musterau.de"))
	if len(got) != 1 || got[0].Host != "fonts.gstatic.com" {
		t.Fatalf("before consent = %+v", got)
	}
}

// A Matomo on the authority's own servers sends nothing anywhere. It is still recorded,
// but it must not be counted among the transfers.
func TestSelfHostedMatomoIsMarked(t *testing.T) {
	pages := []model.PageResult{{Contacts: []model.Contact{
		{Host: "matomo.berlin.de", Phase: model.PhaseBeforeConsent, Requests: 1},
		{Host: "musterau.matomo.cloud", Phase: model.PhaseBeforeConsent, Requests: 1},
	}}}

	for _, obs := range thirdparty.Summarize(pages, "stadt-musterau.de") {
		switch obs.Host {
		case "matomo.berlin.de":
			if !obs.SelfHosted {
				t.Error("a Matomo under a public body's domain is self-hosted")
			}
		case "musterau.matomo.cloud":
			if obs.SelfHosted {
				t.Error("matomo.cloud is not self-hosted")
			}
		}
	}
}
