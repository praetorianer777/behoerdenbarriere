package crawler

import (
	"net/url"
	"path"
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// Dateiendungen, die kein HTML sind. Chrome würde ein PDF zwar laden, axe fände
// darin aber nichts — und der Download kostet die Behörde unnötig Bandbreite.
var skipExtensions = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".zip": true, ".rar": true, ".7z": true,
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".svg": true,
	".webp": true, ".ico": true, ".mp3": true, ".mp4": true, ".avi": true,
	".mov": true, ".wmv": true, ".css": true, ".js": true, ".xml": true,
	".rss": true, ".json": true, ".ics": true, ".txt": true, ".csv": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
}

// Parameter, die nur Sitzung oder Herkunft tragen und denselben Inhalt unter
// beliebig vielen URLs erscheinen lassen.
var dropParams = map[string]bool{
	"utm_source": true, "utm_medium": true, "utm_campaign": true,
	"utm_term": true, "utm_content": true, "fbclid": true, "gclid": true,
	"sid": true, "phpsessid": true, "jsessionid": true,
}

// Normalize bringt eine URL auf eine kanonische Form, damit dieselbe Seite nicht
// mehrfach geprüft wird. Leerer Rückgabewert heißt: nicht crawlbar.
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

// SameSite prüft, ob zwei URLs zur selben registrierbaren Domain gehören.
// Behörden verteilen ihre Auftritte über Subdomains (www., service., formulare.),
// deshalb reicht der Hostvergleich nicht.
func SameSite(a, b string) bool {
	da, err := registrableDomain(a)
	if err != nil {
		return false
	}
	db, err := registrableDomain(b)
	if err != nil {
		return false
	}
	return da == db
}

func registrableDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return publicsuffix.EffectiveTLDPlusOne(u.Hostname())
}

// Pfadbestandteile, die eine Seite für die Prüfung besonders interessant machen:
// die gesetzlich vorgeschriebene Erklärung zur Barrierefreiheit, die
// Kontaktmöglichkeiten und alles, was Nutzer ausfüllen müssen.
var priorityMarkers = []string{
	"barrierefreiheit", "barrierefrei", "accessibility",
	"leichte-sprache", "leichtesprache", "einfache-sprache",
	"gebaerdensprache", "gebärdensprache", "dgs",
	"kontakt", "impressum", "formular", "antrag", "antraege", "anträge",
	"dienstleistung", "buergerservice", "bürgerservice", "onlinedienste",
	"suche", "hilfe", "feedback",
}

// IsPriority erkennt Seiten, die im Score stärker zählen.
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
