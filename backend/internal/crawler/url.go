package crawler

import (
	"net/url"
	"path"
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// Extensions that are not HTML. Chrome would happily load a PDF, but axe finds
// nothing in it — and the download costs the authority bandwidth for nothing.
var skipExtensions = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".zip": true, ".rar": true, ".7z": true,
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".svg": true,
	".webp": true, ".ico": true, ".mp3": true, ".mp4": true, ".avi": true,
	".mov": true, ".wmv": true, ".css": true, ".js": true, ".xml": true,
	".rss": true, ".json": true, ".ics": true, ".txt": true, ".csv": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
}

// Parameters that only carry a session or a referrer and make the same content show
// up under arbitrarily many URLs.
var dropParams = map[string]bool{
	"utm_source": true, "utm_medium": true, "utm_campaign": true,
	"utm_term": true, "utm_content": true, "fbclid": true, "gclid": true,
	"sid": true, "phpsessid": true, "jsessionid": true,
}

// Normalize brings a URL into a canonical form so the same page is not checked twice.
// An empty return means: not crawlable.
func Normalize(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Fragment = ""
	u.User = nil
	u.Host = strings.ToLower(u.Host)
	u.Host = strings.TrimSuffix(u.Host, ":80")
	u.Host = strings.TrimSuffix(u.Host, ":443")

	if u.Path == "" {
		u.Path = "/"
	} else {
		u.Path = path.Clean(u.Path)
		if strings.HasSuffix(raw, "/") && !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
	}
	if ext := strings.ToLower(path.Ext(u.Path)); skipExtensions[ext] {
		return ""
	}

	if q := u.Query(); len(q) > 0 {
		for key := range q {
			if dropParams[strings.ToLower(key)] {
				q.Del(key)
			}
		}
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			values := q[k]
			sort.Strings(values)
			for _, v := range values {
				parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
			}
		}
		u.RawQuery = strings.Join(parts, "&")
	}
	return u.String()
}

// SameSite reports whether a URL still belongs to the site the crawl started from.
//
// The registrable domain is not the right yardstick in Germany: every federal ministry
// sits under bund.de, so bmi.bund.de and bmf.bund.de share it, and a crawl of the
// interior ministry would wander into the finance ministry. What counts instead is the
// start host without a leading "www." — the candidate has to be that host or a
// subdomain of it. Authorities do spread their site over subdomains (service.,
// formulare.), and those are included.
func SameSite(start, candidate string) bool {
	host := siteHost(start)
	if host == "" {
		return false
	}
	other, err := hostOf(candidate)
	if err != nil || other == "" {
		return false
	}
	return other == host || strings.HasSuffix(other, "."+host)
}

func siteHost(rawURL string) string {
	host, err := hostOf(rawURL)
	if err != nil || host == "" {
		return ""
	}
	// A start page given as www.hannover.de must not exclude hannover.de.
	trimmed := strings.TrimPrefix(host, "www.")
	if suffix, _ := publicsuffix.PublicSuffix(trimmed); trimmed == suffix {
		return host
	}
	return trimmed
}

func hostOf(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return strings.ToLower(u.Hostname()), nil
}

// Path fragments that make a page particularly worth checking: the accessibility
// statement required by law, the ways to get in touch, and everything a citizen has to
// fill in.
var priorityMarkers = []string{
	"barrierefreiheit", "barrierefrei", "accessibility",
	"leichte-sprache", "leichtesprache", "einfache-sprache",
	"gebaerdensprache", "gebärdensprache", "dgs",
	"kontakt", "impressum", "formular", "antrag", "antraege", "anträge",
	"dienstleistung", "buergerservice", "bürgerservice", "onlinedienste",
	"suche", "hilfe", "feedback",
}

// IsPriority spots the pages that weigh more in the score.
func IsPriority(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hay := strings.ToLower(u.Path + "?" + u.RawQuery)
	for _, marker := range priorityMarkers {
		if strings.Contains(hay, marker) {
			return true
		}
	}
	return false
}
