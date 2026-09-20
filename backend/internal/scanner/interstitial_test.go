package scanner_test

import (
	"strings"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
)

// Alle drei Fälle sind echt und standen so im Ranking.
func TestInterstitialRecognisesWhatWeActuallyMet(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		title    string
		text     string
		domNodes int
	}{
		{"Saarland", 403, "Establishing a secure connection ...", "", 16},
		{"Thüringen", 200, "Link11 - CAPTCHA", "Bitte bestätigen Sie", 55},
		{"BMWK", 200, "Radware Captcha Page", "", 57},
		{"Cloudflare", 200, "Just a moment...", "", 40},
		{"Akamai", 200, "Access Denied", "Reference #18.1234", 30},
	}
	for _, c := range cases {
		got := scanner.Interstitial(c.status, c.title, c.text, c.domNodes)
		if got == "" {
			t.Errorf("%s: not recognised", c.name)
		}
	}
}

// Der teurere Fehler wäre, eine echte Behördenseite für eine Sperrseite zu halten:
// Dann verschwindet sie aus dem Ranking, und niemand merkt, dass sie geprüft gehört.
func TestInterstitialLeavesRealPagesAlone(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		title    string
		text     string
		domNodes int
	}{
		{"Startseite", 200, "Zoll online - Startseite", strings.Repeat("Inhalt ", 200), 3274},
		{"Fehlerseite der Behörde", 404, "Seite nicht gefunden", "", 800},
		// Eine Seite, die über Bot-Schutz schreibt, ist keine Sperrseite.
		{"Artikel über Cloudflare", 200, "IT-Sicherheit: Was Cloudflare macht", "cloudflare", 2200},
		// Eine kleine Seite allein reicht nicht als Beweis.
		{"karge Seite ohne Anbieter", 200, "Impressum", "Kontakt", 90},
	}
	for _, c := range cases {
		if got := scanner.Interstitial(c.status, c.title, c.text, c.domNodes); got != "" {
			t.Errorf("%s: wrongly taken for bot protection: %q", c.name, got)
		}
	}
}

// Die Begründung landet in der Datenbank und später vor Augen: Sie muss sagen, was los
// war, nicht nur dass etwas los war.
func TestInterstitialSaysWhat(t *testing.T) {
	got := scanner.Interstitial(200, "Link11 - CAPTCHA", "", 55)
	if !strings.Contains(got, "Link11") {
		t.Errorf("reason = %q, want the name in it", got)
	}
	if got := scanner.Interstitial(403, "Establishing a secure connection ...", "", 16); !strings.Contains(got, "403") &&
		!strings.Contains(strings.ToLower(got), "forbidden") {
		t.Errorf("reason = %q, want the status in it", got)
	}
}
