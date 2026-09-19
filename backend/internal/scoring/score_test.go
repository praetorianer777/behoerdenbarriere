package scoring

import (
	"math"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func v(rule string, impact model.Impact, principle model.Principle, nodes int) model.Violation {
	return model.Violation{RuleID: rule, Impact: impact, Principle: principle, NodeCount: nodes}
}

func page(url string, nodes int, entry, priority bool, vs ...model.Violation) model.PageResult {
	return model.PageResult{URL: url, DOMNodes: nodes, IsEntry: entry, Priority: priority, Violations: vs}
}

func TestPageScoreCleanPageIsPerfect(t *testing.T) {
	if got := PageScore(page("/", 800, true, false)); got != 100 {
		t.Fatalf("saubere Seite: got %v, want 100", got)
	}
}

func TestPageScoreOrdering(t *testing.T) {
	clean := PageScore(page("/", 800, true, false))
	minor := PageScore(page("/", 800, true, false, v("r1", model.ImpactMinor, model.Robust, 1)))
	critical := PageScore(page("/", 800, true, false, v("r1", model.ImpactCritical, model.Perceivable, 1)))
	many := PageScore(page("/", 800, true, false, v("r1", model.ImpactCritical, model.Perceivable, 40)))

	if !(clean > minor && minor > critical && critical > many) {
		t.Fatalf("Reihenfolge verletzt: clean=%v minor=%v critical=%v many=%v", clean, minor, critical, many)
	}
	if many < 0 || clean > 100 {
		t.Fatalf("Score außerhalb 0..100: %v / %v", many, clean)
	}
}

func TestPageScoreNormalizesBySize(t *testing.T) {
	small := PageScore(page("/", 200, true, false, v("r1", model.ImpactSerious, model.Operable, 1)))
	large := PageScore(page("/", 4000, true, false, v("r1", model.ImpactSerious, model.Operable, 1)))
	if large <= small {
		t.Fatalf("derselbe Verstoß wiegt auf der großen Seite nicht leichter: small=%v large=%v", small, large)
	}
}

func TestPageScoreTinyPageIsNotOverPenalized(t *testing.T) {
	tiny := PageScore(page("/", 3, true, false, v("r1", model.ImpactMinor, model.Robust, 1)))
	floored := PageScore(page("/", minDOMNodes, true, false, v("r1", model.ImpactMinor, model.Robust, 1)))
	if math.Abs(tiny-floored) > 0.001 {
		t.Fatalf("Mindestgröße greift nicht: tiny=%v floored=%v", tiny, floored)
	}
}

func TestSiteScoreWeightsEntryPageHigher(t *testing.T) {
	bad := v("r1", model.ImpactCritical, model.Perceivable, 20)

	badEntry := SiteScore([]model.PageResult{
		page("/", 800, true, false, bad),
		page("/a", 800, false, false),
		page("/b", 800, false, false),
	})
	badSubpage := SiteScore([]model.PageResult{
		page("/", 800, true, false),
		page("/a", 800, false, false, bad),
		page("/b", 800, false, false),
	})
	if badEntry.Score >= badSubpage.Score {
		t.Fatalf("Startseite wiegt nicht schwerer: entry=%v subpage=%v", badEntry.Score, badSubpage.Score)
	}
}

func TestSiteScoreIgnoresFailedPages(t *testing.T) {
	failed := model.PageResult{URL: "/kaputt", Err: "timeout"}
	withFailure := SiteScore([]model.PageResult{page("/", 800, true, false), failed})
	if withFailure.Score != 100 {
		t.Fatalf("fehlgeschlagene Seite fließt in den Score ein: %v", withFailure.Score)
	}
	if withFailure.Pages != 1 {
		t.Fatalf("Seitenzahl falsch: %d", withFailure.Pages)
	}
}

func TestSiteScoreNoUsablePages(t *testing.T) {
	res := SiteScore([]model.PageResult{{URL: "/x", Err: "dns"}})
	if res.Score != 0 || res.Grade != "" || res.Pages != 0 {
		t.Fatalf("leeres Ergebnis erwartet, got %+v", res)
	}
}

func TestSiteScorePrincipleSubscores(t *testing.T) {
	res := SiteScore([]model.PageResult{
		page("/", 800, true, false, v("image-alt", model.ImpactCritical, model.Perceivable, 30)),
	})
	if res.Principles[model.Perceivable] >= res.Principles[model.Operable] {
		t.Fatalf("Teilscore trennt die Prinzipien nicht: %+v", res.Principles)
	}
	if res.Principles[model.Operable] != 100 {
		t.Fatalf("unbelastetes Prinzip sollte 100 sein: %v", res.Principles[model.Operable])
	}
}

func TestGrade(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{100, "A"}, {90, "A"}, {89.99, "B"}, {80, "B"}, {70, "C"},
		{60, "D"}, {50, "E"}, {49.99, "F"}, {0, "F"},
	}
	for _, c := range cases {
		if got := Grade(c.score); got != c.want {
			t.Errorf("Grade(%v) = %s, want %s", c.score, got, c.want)
		}
	}
}

func TestPrincipleFromTags(t *testing.T) {
	cases := []struct {
		tags []string
		want model.Principle
	}{
		{[]string{"cat.text-alternatives", "wcag2a", "wcag111"}, model.Perceivable},
		{[]string{"wcag2a", "wcag211"}, model.Operable},
		{[]string{"wcag2aa", "wcag312"}, model.Understandable},
		{[]string{"wcag2a", "wcag412"}, model.Robust},
		{[]string{"best-practice"}, model.Robust},
		{nil, model.Robust},
		{[]string{"wcag2a"}, model.Robust},
	}
	for _, c := range cases {
		if got := Principle(c.tags); got != c.want {
			t.Errorf("Principle(%v) = %s, want %s", c.tags, got, c.want)
		}
	}
}

func TestSummarizeRulesOrdersBySeverityAndCounts(t *testing.T) {
	pages := []model.PageResult{
		page("/", 800, true, false,
			v("color-contrast", model.ImpactSerious, model.Perceivable, 12),
			v("region", model.ImpactModerate, model.Robust, 1),
		),
		page("/a", 800, false, false,
			v("color-contrast", model.ImpactSerious, model.Perceivable, 3),
			v("label", model.ImpactCritical, model.Perceivable, 2),
		),
		{URL: "/kaputt", Err: "timeout"},
	}

	got := SummarizeRules(pages)
	if len(got) != 3 {
		t.Fatalf("erwartet 3 Regeln, got %d", len(got))
	}
	if got[0].RuleID != "label" {
		t.Errorf("schwerster Verstoß nicht zuerst: %s", got[0].RuleID)
	}
	if got[1].RuleID != "color-contrast" || got[1].Pages != 2 || got[1].Nodes != 15 {
		t.Errorf("Aggregation falsch: %+v", got[1])
	}
	if got[1].Sample == nil {
		t.Error("Beispiel fehlt")
	}
}
