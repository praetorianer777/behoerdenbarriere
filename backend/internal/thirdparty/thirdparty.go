// Package thirdparty turns the host names a page contacted into something a reader can
// judge: which company or service they belong to, whether the receiver is itself a
// public body, and whether anything was sent before the visitor could object.
//
// The classification is deliberately kept out of the database. A host name is stored
// as it was observed; how we read it may improve, and a better reading must not require
// crawling every authority again.
package thirdparty

import (
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// Group is the name people recognise a host by. It is an attribution of the host, not
// of the data: that fonts.gstatic.com belongs to Google is a fact about the name; what
// Google does with the request is not something a crawler can see.
type Group string

const (
	GoogleFonts     Group = "google-fonts"
	GoogleAnalytics Group = "google-analytics"
	GoogleMaps      Group = "google-maps"
	GoogleAds       Group = "google-ads"
	GoogleOther     Group = "google-other"
	YouTube         Group = "youtube"
	Vimeo           Group = "vimeo"
	Meta            Group = "meta"
	X               Group = "x"
	LinkedIn        Group = "linkedin"
	Matomo          Group = "matomo"
	Etracker        Group = "etracker"
	ConsentTool     Group = "consent-tool"
	CDN             Group = "cdn"
	Unknown         Group = "unknown"
)

// rules are checked in order, first match wins. The order matters: youtube.com would
// otherwise be caught by the catch-all for Google's domains, and the two are worth
// telling apart — an embedded video is a decision someone made about a page, a font is
// usually an accident of a theme.
var rules = []struct {
	domains []string
	group   Group
}{
	{[]string{"fonts.googleapis.com", "fonts.gstatic.com"}, GoogleFonts},
	{[]string{"google-analytics.com", "googletagmanager.com", "analytics.google.com",
		"ssl.google-analytics.com"}, GoogleAnalytics},
	{[]string{"maps.googleapis.com", "maps.google.com", "maps.gstatic.com",
		"khms0.google.com", "khms1.google.com"}, GoogleMaps},
	{[]string{"doubleclick.net", "googleadservices.com", "googlesyndication.com",
		"adservice.google.com", "google.com/ads"}, GoogleAds},
	{[]string{"youtube.com", "youtube-nocookie.com", "ytimg.com", "youtu.be",
		"googlevideo.com"}, YouTube},
	{[]string{"vimeo.com", "vimeocdn.com", "player.vimeo.com"}, Vimeo},
	{[]string{"facebook.com", "facebook.net", "fbcdn.net", "instagram.com",
		"whatsapp.com"}, Meta},
	{[]string{"twitter.com", "x.com", "twimg.com", "t.co"}, X},
	{[]string{"linkedin.com", "licdn.com"}, LinkedIn},
	{[]string{"matomo.cloud", "matomo.org", "piwik.pro"}, Matomo},
	{[]string{"etracker.com", "etracker.de"}, Etracker},
	{[]string{"cookiebot.com", "usercentrics.eu", "onetrust.com", "cookielaw.org",
		"consentmanager.net", "borlabs.io", "cookiefirst.com", "klaro.org",
		"privacy-mgmt.com"}, ConsentTool},
	{[]string{"jsdelivr.net", "unpkg.com", "cdnjs.cloudflare.com", "bootstrapcdn.com",
		"jquery.com", "akamaized.net", "akamaihd.net", "cloudfront.net",
		"fontawesome.com", "typekit.net", "use.typekit.com"}, CDN},
	// Everything else under Google's own domains, after the specific services above.
	{[]string{"google.com", "googleapis.com", "gstatic.com", "google.de",
		"recaptcha.net"}, GoogleOther},
}

// selfHosted recognises an analytics tool a public body runs itself. The software is
// the same, the data protection question is not: a Matomo on the authority's own server
// sends nothing to anyone.
var selfHosted = map[Group]bool{Matomo: true}

// Classify names the service a host belongs to.
func Classify(host string) Group {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, rule := range rules {
		for _, domain := range rule.domains {
			if host == domain || strings.HasSuffix(host, "."+domain) {
				return rule.group
			}
		}
	}
	// Matomo and Piwik are usually recognisable by the host name alone, wherever they
	// run — "matomo.example.de", "piwik.stadt-x.de".
	if strings.Contains(host, "matomo") || strings.Contains(host, "piwik") {
		return Matomo
	}
	return Unknown
}

// publicSuffixes are the domains under which German public bodies publish. The list is
// incomplete by nature — every municipality has its own domain — and is only used to
// soften the picture where it would otherwise mislead: a state portal loading a font
// from a federal server is not the same as one loading it from Google.
var publicSuffixes = []string{
	"bund.de", "bundestag.de", "bundesrat.de", "europa.eu",
	"baden-wuerttemberg.de", "bayern.de", "berlin.de", "brandenburg.de", "bremen.de",
	"hamburg.de", "hessen.de", "mv-regierung.de", "niedersachsen.de", "nrw.de",
	"land.nrw", "rlp.de", "saarland.de", "sachsen.de", "sachsen-anhalt.de",
	"schleswig-holstein.de", "thueringen.de", "govdata.de", "dataport.de",
	"itzbund.de", "ozg-cloud.de",
}

// PublicBody reports whether the receiving host is itself a public body, as far as the
// domain shows it.
func PublicBody(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, suffix := range publicSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

// Registrable is the domain a host belongs to — "www.bmi.bund.de" and "bmi.bund.de"
// share one, "fonts.gstatic.com" does not. Falls back to the host itself where the
// public suffix list has nothing to say.
func Registrable(host string) string {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(host, "."), ":443"))
	if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil || domain == "" {
		return host
	}
	return domain
}

// SameSite reports whether two hosts belong to the same registrable domain.
func SameSite(a, b string) bool {
	return a != "" && b != "" && Registrable(a) == Registrable(b)
}

// Observation is one classified host, ready to be shown.
type Observation struct {
	Host       string             `json:"host"`
	Domain     string             `json:"domain"`
	Group      Group              `json:"group"`
	PublicBody bool               `json:"public_body"`
	SelfHosted bool               `json:"self_hosted"`
	Phase      model.ContactPhase `json:"phase"`
	Requests   int                `json:"requests"`
	Pages      int                `json:"pages"`
}

// Seen is a host as it comes back from the database: already folded over the pages of
// a scan, still unclassified.
type Seen struct {
	Host     string
	Phase    model.ContactPhase
	Requests int
	Pages    int
}

// Summarize folds the contacts of all pages of a scan into one entry per host and
// phase. A host contacted by forty pages is one finding, not forty, but how many pages
// it was stays visible: a tracker on the entry page alone is something else than one on
// every page of the site.
func Summarize(pages []model.PageResult, siteHost string) []Observation {
	type key struct {
		host  string
		phase model.ContactPhase
	}
	folded := map[key]*Seen{}
	order := make([]key, 0)

	for _, page := range pages {
		for _, contact := range page.Contacts {
			k := key{host: contact.Host, phase: contact.Phase}
			seen, ok := folded[k]
			if !ok {
				seen = &Seen{Host: contact.Host, Phase: contact.Phase}
				folded[k] = seen
				order = append(order, k)
			}
			seen.Requests += max(contact.Requests, 1)
			seen.Pages++
		}
	}

	out := make([]Seen, 0, len(folded))
	for _, k := range order {
		out = append(out, *folded[k])
	}
	return Describe(out, siteHost)
}

// Describe classifies what was seen and puts the findings in the order they should be
// read. siteHost is the authority's own host; contacts within its registrable domain
// are dropped — a site loading its own assets says nothing about anyone's data leaving.
func Describe(seen []Seen, siteHost string) []Observation {
	out := make([]Observation, 0, len(seen))
	for _, s := range seen {
		if s.Host == "" || SameSite(s.Host, siteHost) {
			continue
		}
		group := Classify(s.Host)
		out = append(out, Observation{
			Host:       s.Host,
			Domain:     Registrable(s.Host),
			Group:      group,
			PublicBody: PublicBody(s.Host),
			SelfHosted: selfHosted[group] && PublicBody(s.Host),
			Phase:      s.Phase,
			Requests:   s.Requests,
			Pages:      s.Pages,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Phase != out[j].Phase {
			// Before consent first: that is the part with legal weight.
			return phaseRank(out[i].Phase) < phaseRank(out[j].Phase)
		}
		if out[i].Pages != out[j].Pages {
			return out[i].Pages > out[j].Pages
		}
		return out[i].Host < out[j].Host
	})
	return out
}

func phaseRank(p model.ContactPhase) int {
	switch p {
	case model.PhaseBeforeConsent:
		return 0
	case model.PhaseAfterDeclined:
		return 1
	default:
		return 2
	}
}

// BeforeConsent picks out what left the browser before anyone could object, leaving out
// receivers that are public bodies themselves.
func BeforeConsent(observations []Observation) []Observation {
	out := make([]Observation, 0)
	for _, obs := range observations {
		if obs.Phase == model.PhaseBeforeConsent && !obs.PublicBody {
			out = append(out, obs)
		}
	}
	return out
}
