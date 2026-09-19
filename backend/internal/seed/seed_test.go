package seed

import (
	"strings"
	"testing"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func TestParseValidEntry(t *testing.T) {
	got, err := Parse([]byte(`
- {slug: bmi, name: Bundesministerium des Innern, url: "https://www.bmi.bund.de", level: bund, category: ministerium}
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	a := got[0]
	if a.Slug != "bmi" || a.Level != model.LevelBund || !a.Active {
		t.Errorf("fields not carried over: %+v", a)
	}
	// A trailing slash is added so that the same start page does not appear as two
	// different URLs in the crawler.
	if a.URL != "https://www.bmi.bund.de/" {
		t.Errorf("URL = %q", a.URL)
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"empty list":     `[]`,
		"missing slug":   `- {name: Amt, url: "https://a.de", level: bund}`,
		"missing name":   `- {slug: amt, url: "https://a.de", level: bund}`,
		"duplicate slug": "- {slug: a, name: A, url: \"https://a.de\", level: bund}\n- {slug: a, name: B, url: \"https://b.de\", level: bund}",
		"unknown level":  `- {slug: a, name: A, url: "https://a.de", level: gemeinde}`,
		"state missing":  `- {slug: a, name: A, url: "https://a.de", level: kommune}`,
		"plain http":     `- {slug: a, name: A, url: "http://a.de", level: bund}`,
		"not a URL":      `- {slug: a, name: A, url: "a.de", level: bund}`,
		"broken yaml":    `- {slug: a`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(raw)); err == nil {
				t.Fatalf("accepted: %s", raw)
			}
		})
	}
}

// The error has to name the entry, otherwise a typo in a list of a hundred means
// searching by hand.
func TestParseErrorNamesEntry(t *testing.T) {
	_, err := Parse([]byte("- {slug: a, name: A, url: \"https://a.de\", level: bund}\n- {slug: b, name: B, url: \"a.de\", level: bund}"))
	if err == nil {
		t.Fatal("no error")
	}
	if !strings.Contains(err.Error(), "entry 2") || !strings.Contains(err.Error(), "b") {
		t.Fatalf("error does not name the entry: %v", err)
	}
}

func TestEmbeddedSeedsAreValid(t *testing.T) {
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) < 50 {
		t.Fatalf("only %d authorities on the list", len(got))
	}

	states := map[string]bool{}
	levels := map[model.Level]int{}
	for _, a := range got {
		levels[a.Level]++
		if a.Level == model.LevelLand {
			states[a.State] = true
		}
	}
	// All sixteen states have to be on the list; a missing one would look like a
	// state without a web presence in the nationwide comparison.
	if len(states) != 16 {
		t.Errorf("%d states as state portals, want 16", len(states))
	}
	if levels[model.LevelBund] < 20 || levels[model.LevelKommune] < 20 {
		t.Errorf("distribution across levels too thin: %v", levels)
	}
}
