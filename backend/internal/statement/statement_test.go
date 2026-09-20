package statement

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Wording close to what authorities actually publish; the model declaration in
// Implementing Decision (EU) 2018/1523 is what most of them copy.
const complete = `Erklärung zur Barrierefreiheit

Das Bundesministerium ist bemüht, seine Website im Einklang mit § 12b BGG barrierefrei
zugänglich zu machen. Diese Website ist mit der Barrierefreie-Informationstechnik-
Verordnung teilweise vereinbar. Die Unvereinbarkeiten sind nachfolgend aufgeführt.

Nicht barrierefreie Inhalte: Einige ältere PDF-Dokumente sind nicht barrierefrei.
Videos aus dem Archiv haben keine Untertitel.

Diese Erklärung wurde am 14.03.2026 erstellt und zuletzt am 02.09.2026 überprüft.

Barrieren melden: Sind Ihnen Mängel bei der barrierefreien Nutzung aufgefallen?
Schreiben Sie uns an barrierefreiheit@example.bund.de.

Schlichtungsverfahren: Wenn auch die Antwort nicht zufriedenstellend ausfällt, können
Sie die Schlichtungsstelle nach § 16 BGG anrufen.`

func statementPage(text string, depth int) Input {
	return Input{Pages: []Page{
		{URL: "https://www.example.bund.de/", Text: "Startseite", Depth: 0},
		{URL: "https://www.example.bund.de/erklaerung-zur-barrierefreiheit", Text: text, Depth: depth},
	}}
}

func met(t *testing.T, result Result) map[Requirement]bool {
	t.Helper()
	out := map[Requirement]bool{}
	for _, finding := range result.Findings {
		out[finding.Requirement] = finding.Met
	}
	return out
}

func TestCompleteStatement(t *testing.T) {
	result := Check(statementPage(complete, 1))

	if !result.Found() {
		t.Fatal("the statement was not found")
	}
	if !result.Complete() {
		t.Fatalf("a complete statement was marked incomplete: %+v", met(t, result))
	}
	if result.Met() != len(Requirements) {
		t.Errorf("%d of %d requirements met", result.Met(), len(Requirements))
	}
}

func TestMissingStatement(t *testing.T) {
	result := Check(Input{Pages: []Page{
		{URL: "https://www.example.bund.de/", Text: "Startseite", Depth: 0},
		{URL: "https://www.example.bund.de/impressum", Text: "Impressum", Depth: 1},
	}})

	if result.State != StateMissing || result.Complete() || result.Met() != 0 {
		t.Fatalf("something was found where there is nothing: %+v", result)
	}
	// Every requirement is still listed, so the page can say what is missing rather
	// than only that something is.
	if len(result.Findings) != len(Requirements) {
		t.Fatalf("%d findings, want %d", len(result.Findings), len(Requirements))
	}
}

func TestEachRequirementCanFailOnItsOwn(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		missing Requirement
	}{
		{
			"without a conformance status",
			strings.Replace(complete, "Diese Website ist mit der Barrierefreie-Informationstechnik-\nVerordnung teilweise vereinbar.", "", 1),
			Conformance,
		},
		{
			"without the shortcomings",
			strings.Replace(complete, "Nicht barrierefreie Inhalte: Einige ältere PDF-Dokumente sind nicht barrierefrei.", "", 1),
			Shortcomings,
		},
		{
			"without a date",
			strings.Replace(complete, "Diese Erklärung wurde am 14.03.2026 erstellt und zuletzt am 02.09.2026 überprüft.", "", 1),
			Date,
		},
		{
			"without a way to report a barrier",
			strings.Replace(complete, "Barrieren melden: Sind Ihnen Mängel bei der barrierefreien Nutzung aufgefallen?\nSchreiben Sie uns an barrierefreiheit@example.bund.de.", "", 1),
			Feedback,
		},
		{
			"without the enforcement procedure",
			strings.Replace(complete, "Schlichtungsverfahren: Wenn auch die Antwort nicht zufriedenstellend ausfällt, können\nSie die Schlichtungsstelle nach § 16 BGG anrufen.", "", 1),
			Enforcement,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := Check(statementPage(c.text, 1))
			if !result.Found() {
				t.Fatal("the statement was not found")
			}
			byRequirement := met(t, result)
			if byRequirement[c.missing] {
				t.Errorf("%s was reported as met", c.missing)
			}
			for requirement, ok := range byRequirement {
				if requirement != c.missing && !ok {
					t.Errorf("%s failed as well, although only %s was removed", requirement, c.missing)
				}
			}
		})
	}
}

// A statement buried three clicks deep does not meet what the law asks — it has to be
// reachable from the start page.
func TestStatementBuriedDeep(t *testing.T) {
	result := Check(statementPage(complete, 3))
	if met(t, result)[Reachable] {
		t.Fatal("a statement three clicks away was counted as reachable")
	}
	if result.Complete() {
		t.Fatal("marked complete although it is out of reach")
	}
}

// An announcement to report barriers without any way to do so is not a feedback
// mechanism.
func TestFeedbackNeedsAWayToReach(t *testing.T) {
	text := strings.Replace(complete, "Schreiben Sie uns an barrierefreiheit@example.bund.de.", "Wir freuen uns über Ihre Rückmeldung.", 1)

	if met(t, Check(statementPage(text, 1)))[Feedback] {
		t.Fatal("an invitation without an address counted as a feedback mechanism")
	}
}

// The page named as the statement wins over one that merely talks about
// accessibility — bfarm.de has a page about the accessibility of a code register
// whose URL is longer than the real statement's, and picking the longest URL read the
// wrong page.
func TestPrefersThePageNamedAsTheStatement(t *testing.T) {
	pages := Input{Pages: []Page{
		{URL: "https://www.example.bund.de/Kodiersysteme/Services/OID-Register/Barrierefreiheit-OID",
			Text: "Barrierefreiheit des OID-Registers.", Depth: 2},
		{URL: "https://www.example.bund.de/erklaerung-zur-barrierefreiheit", Text: complete, Depth: 1},
	}}

	result := Check(pages)
	if !strings.HasSuffix(result.URL, "erklaerung-zur-barrierefreiheit") {
		t.Fatalf("chosen page = %s", result.URL)
	}
	if !result.Complete() {
		t.Fatalf("the statement was not read: %+v", met(t, result))
	}
}

// Among pages that are equally named, the one closer to the start page wins.
func TestPrefersTheShallowerPage(t *testing.T) {
	pages := Input{Pages: []Page{
		{URL: "https://www.example.bund.de/service/a/b/barrierefreiheit", Text: "Kurzer Hinweis.", Depth: 3},
		{URL: "https://www.example.bund.de/barrierefreiheit", Text: complete, Depth: 1},
	}}
	if result := Check(pages); !strings.HasSuffix(result.URL, "de/barrierefreiheit") {
		t.Fatalf("chosen page = %s", result.URL)
	}
}

// The evidence is what makes the verdict arguable, so it has to be there.
func TestFindingsCarryEvidence(t *testing.T) {
	result := Check(statementPage(complete, 1))
	for _, finding := range result.Findings {
		if finding.Requirement == Reachable {
			continue
		}
		if finding.Met && finding.Evidence == "" {
			t.Errorf("%s is met but says nothing about why", finding.Requirement)
		}
	}
}

func TestIsStatementURL(t *testing.T) {
	yes := []string{
		"https://a.de/erklaerung-zur-barrierefreiheit",
		"https://a.de/DE/Service/Barrierefreiheit/barrierefreiheit_node.html",
		"https://a.de/barrierefreiheitserklaerung",
		"https://a.de/en/accessibility-statement",
	}
	for _, url := range yes {
		if !IsStatementURL(url) {
			t.Errorf("IsStatementURL(%q) = false", url)
		}
	}

	no := []string{"https://a.de/impressum", "https://a.de/kontakt", "https://a.de/presse/2026"}
	for _, url := range no {
		if IsStatementURL(url) {
			t.Errorf("IsStatementURL(%q) = true", url)
		}
	}
}

// Seen but not readable is not the same as absent. The RKI links its statement and
// excludes the directory it lives in from crawlers; saying it has none would accuse
// an authority that has one.
func TestLinkedButUnreadable(t *testing.T) {
	result := Check(Input{
		Pages: []Page{{URL: "https://www.rki.de/", Text: "Startseite", Depth: 0}},
		SeenLinks: []string{
			"https://www.rki.de/impressum",
			"https://www.rki.de/DE/Service/Barrierefreiheit/barrierefreiheit_node.html",
		},
	})

	if result.State != StateUnreadable {
		t.Fatalf("state = %s, want unreadable", result.State)
	}
	if result.Found() || result.Complete() {
		t.Error("an unread statement was counted as read")
	}
	if result.URL == "" {
		t.Error("the address we could not read is missing")
	}
	// The requirements are still listed, all unmet — nothing is claimed about them.
	if len(result.Findings) != len(Requirements) || result.Met() != 0 {
		t.Errorf("findings = %+v", result.Findings)
	}
}

func TestNoStatementLinkAtAll(t *testing.T) {
	result := Check(Input{
		Pages:     []Page{{URL: "https://a.de/", Text: "Startseite"}},
		SeenLinks: []string{"https://a.de/impressum", "https://a.de/kontakt"},
	})
	if result.State != StateMissing {
		t.Fatalf("state = %s, want missing", result.State)
	}
}

// Die Erklärung des Bundespräsidenten zeigte bei uns „…barrierefrei�n": Das Fenster um
// den Treffer wurde in Bytes bemessen und schnitt einen Umlaut mitten durch. Der
// Nachweis ist genau das, woran sich unser Urteil prüfen lassen soll — kaputt taugt er
// dafür nicht.
//
// Entscheidend ist, dass der Abstand zwischen den mehrbytigen Zeichen und dem Treffer
// wächst: Verschiebt man den ganzen Text, wandert die Fenstergrenze mit und trifft nie
// in ein Zeichen hinein — ein Test in dieser Form bleibt grün, während der Fehler da
// ist. Das war mein erster Versuch.
func TestEvidenceIsNeverCutInsideACharacter(t *testing.T) {
	umlaute := strings.Repeat("ö", 40) + "„Gebärdensprache“ — "

	for abstand := range 60 {
		text := umlaute + strings.Repeat("a", abstand) +
			` Diese Website ist mit der BITV 2.0 teilweise vereinbar. Nicht barrierefrei ` +
			`sind Gebärdensprachvideos und ältere PDF-Dokumente. Erstellt am 14.03.2026, ` +
			`Barrieren melden Sie an barrierefreiheit@example.de, ` +
			`Schlichtungsstelle nach § 16 BGG.`

		result := Check(Input{
			Pages: []Page{{
				URL: "https://example.de/erklaerung-zur-barrierefreiheit", Text: text,
			}},
		})

		for _, finding := range result.Findings {
			// ValidString, nicht ContainsRune: Ein halbes Zeichen ist ein ungültiges
			// Byte, kein Ersatzzeichen. Als Ersatzzeichen sieht es erst der Browser.
			if !utf8.ValidString(finding.Evidence) {
				t.Fatalf("Abstand %d: %s hat ein zerschnittenes Zeichen: %q",
					abstand, finding.Requirement, finding.Evidence)
			}
		}
	}
}
