package crawler

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"fragment dropped", "https://www.bmi.bund.de/start#inhalt", "https://www.bmi.bund.de/start"},
		{"empty path becomes root", "https://www.bmi.bund.de", "https://www.bmi.bund.de/"},
		{"host lowercased", "https://WWW.BMI.Bund.DE/Start", "https://www.bmi.bund.de/Start"},
		{"default port dropped", "https://www.bmi.bund.de:443/start", "https://www.bmi.bund.de/start"},
		{"tracking parameter dropped", "https://a.de/x?utm_source=news&id=7", "https://a.de/x?id=7"},
		{"session id dropped", "https://a.de/x?jsessionid=abc", "https://a.de/x"},
		{"parameters sorted", "https://a.de/x?b=2&a=1", "https://a.de/x?a=1&b=2"},
		{"path cleaned", "https://a.de/a/../b/./c", "https://a.de/b/c"},
		{"trailing slash kept", "https://a.de/amt/", "https://a.de/amt/"},
		{"credentials dropped", "https://user:pw@a.de/x", "https://a.de/x"},

		{"mailto rejected", "mailto:amt@a.de", ""},
		{"javascript rejected", "javascript:void(0)", ""},
		{"pdf rejected", "https://a.de/formular.pdf", ""},
		{"image rejected", "https://a.de/wappen.PNG", ""},
		{"stylesheet rejected", "https://a.de/style.css", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Normalize(c.in); got != c.want {
				t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// The same page must not enter the frontier under two spellings — the page budget
// would go on duplicates.
func TestNormalizeIsStable(t *testing.T) {
	variants := []string{
		"https://www.bmi.bund.de/start?b=2&a=1#oben",
		"https://WWW.bmi.bund.de:443/start?a=1&b=2",
		"https://www.bmi.bund.de/x/../start?b=2&a=1&utm_medium=mail",
	}
	first := Normalize(variants[0])
	for _, v := range variants[1:] {
		if got := Normalize(v); got != first {
			t.Errorf("Normalize(%q) = %q, want %q", v, got, first)
		}
	}
}

func TestSameSite(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		// Authorities spread their site over subdomains, and those belong to the site.
		{"https://www.bmi.bund.de/", "https://service.bmi.bund.de/formular", true},
		{"https://www.hannover.de/", "https://www.hannover.de/kontakt", true},
		// Every federal ministry sits under bund.de, so the registrable domain is
		// shared and must not be what decides.
		{"https://www.bmi.bund.de/", "https://www.bmf.bund.de/", false},
		{"https://www.bmi.bund.de/", "https://www.bsi.bund.de/", false},
		// The start page is usually given with www., the links come without it.
		{"https://www.hannover.de/", "https://hannover.de/kontakt", true},
		{"https://hannover.de/", "https://www.hannover.de/kontakt", true},
		{"https://www.bmi.bund.de/", "https://twitter.com/bmi", false},
		{"https://www.bmi.bund.de/", "not a url", false},
	}
	for _, c := range cases {
		if got := SameSite(c.a, c.b); got != c.want {
			t.Errorf("SameSite(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestIsPriority(t *testing.T) {
	priority := []string{
		"https://a.de/erklaerung-zur-barrierefreiheit",
		"https://a.de/service/leichte-sprache",
		"https://a.de/gebaerdensprache/",
		"https://a.de/kontakt",
		"https://a.de/impressum",
		"https://a.de/buergerservice/antrag-wohngeld",
		"https://a.de/formulare/anmeldung",
		"https://a.de/suche?q=pass",
	}
	for _, u := range priority {
		if !IsPriority(u) {
			t.Errorf("IsPriority(%q) = false", u)
		}
	}

	ordinary := []string{
		"https://a.de/",
		"https://a.de/presse/mitteilung-2026-03",
		"https://a.de/aktuelles",
	}
	for _, u := range ordinary {
		if IsPriority(u) {
			t.Errorf("IsPriority(%q) = true", u)
		}
	}
}
