package scanner

import (
	"net/http"
	"strings"
)

// Interstitial recognises a page that stands in front of a website instead of being
// one: a captcha, a bot check, a block page.
//
// It matters because such a page gets scored otherwise, and the number is then a
// verdict about our own crawler published under someone else's name. Both directions
// happened: Saarland received 0.00 and an F for a 403 block page, the BMWK 100 and an A
// for a Radware captcha.
//
// Returned is a short reason, or an empty string when the page looks like a website.
func Interstitial(status int, title, text string, domNodes int) string {
	switch status {
	case http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return "Zugriff abgewiesen (HTTP " + http.StatusText(status) + ")"
	}

	lowerTitle := strings.ToLower(title)
	for _, phrase := range titleMarkers {
		if strings.Contains(lowerTitle, phrase) {
			return "Bot-Schutz: " + strings.TrimSpace(title)
		}
	}

	// Der Name eines Anbieters allein reicht nicht: Eine echte Seite darf über
	// Cloudflare schreiben. Zusammen mit einer Seite, die fast nichts enthält, ist es
	// eindeutig genug — eine Behördenstartseite hat Hunderte von Knoten.
	if domNodes < interstitialNodes {
		haystack := strings.ToLower(title + " " + text)
		for _, vendor := range vendorMarkers {
			if strings.Contains(haystack, vendor) {
				return "Bot-Schutz: " + vendor
			}
		}
	}
	return ""
}

// interstitialNodes: Unterhalb dieser Größe ist eine Startseite keine Startseite. Die
// beobachteten Sperrseiten hatten 16, 55 und 57 Knoten; die kleinste echte
// Behördenstartseite in unserem Bestand liegt bei mehreren hundert.
const interstitialNodes = 300

// titleMarkers sind für sich genommen aussagekräftig — so nennt keine Behörde ihre
// Startseite.
var titleMarkers = []string{
	"captcha",
	"just a moment",
	"attention required",
	"security check",
	"sicherheitsüberprüfung",
	"establishing a secure connection",
	"access denied",
	"zugriff verweigert",
	"bot management",
	"are you a robot",
	"checking your browser",
	"ddos protection",
}

var vendorMarkers = []string{
	"radware",
	"link11",
	"cloudflare",
	"imperva",
	"incapsula",
	"akamai",
	"perfdrive",
	"sucuri",
	"datadome",
	"queue-it",
}
