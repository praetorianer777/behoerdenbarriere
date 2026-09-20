package wikidata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func row(item, label, website string) binding {
	return binding{
		"item":      {Value: item},
		"itemLabel": {Value: label},
		"website":   {Value: website},
	}
}

func TestAgenciesMapsARow(t *testing.T) {
	got := Agencies([]binding{
		row("http://www.wikidata.org/entity/Q10523", "Landkreis Miesbach", "https://www.landkreis-miesbach.de/"),
	})

	if len(got) != 1 {
		t.Fatalf("%d entries", len(got))
	}
	a := got[0]
	if a.Slug != "landkreis-miesbach" || a.Name != "Landkreis Miesbach" {
		t.Errorf("entry = %+v", a)
	}
	if a.Level != model.LevelKreis || a.Source != model.SourceWikidata || a.ExternalID != "Q10523" {
		t.Errorf("provenance = %+v", a)
	}
}

// An import that quietly keeps unusable rows would look complete in the ranking while
// pointing nowhere.
func TestAgenciesDropsWhatCannotBeUsed(t *testing.T) {
	got := Agencies([]binding{
		row("http://www.wikidata.org/entity/Q1", "", "https://a.de/"),
		row("http://www.wikidata.org/entity/Q2", "Landkreis Ohnehin", ""),
		row("", "Landkreis Namenlos", "https://b.de/"),
		row("http://www.wikidata.org/entity/Q3", "Landkreis Falsch", "ftp://c.de/"),
		// Wikidata has no German label for it; an entry called "Q4" helps nobody.
		row("http://www.wikidata.org/entity/Q4", "Q4", "https://d.de/"),
	})

	if len(got) != 0 {
		t.Fatalf("unusable rows were kept: %+v", got)
	}
}

func TestAgenciesKeepsEachEntityOnce(t *testing.T) {
	got := Agencies([]binding{
		row("http://www.wikidata.org/entity/Q10523", "Landkreis Miesbach", "https://www.landkreis-miesbach.de/"),
		row("http://www.wikidata.org/entity/Q10523", "Landkreis Miesbach", "https://www.landkreis-miesbach.de/en"),
	})
	if len(got) != 1 {
		t.Fatalf("%d entries for one district", len(got))
	}
}

// Wikidata holds plenty of http addresses that have long redirected, and a scan that
// starts on http counts a redirect as the page.
func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"http://www.kreis-steinfurt.de":           "https://www.kreis-steinfurt.de/",
		"https://www.landkreis-leer.de/start?x=1": "https://www.landkreis-leer.de/start",
		"https://a.de/x#unten":                    "https://a.de/x",
		"ftp://a.de/":                             "",
		"kein-url":                                "",
	}
	for in, want := range cases {
		if got := normalizeURL(in); got != want {
			t.Errorf("normalizeURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Landkreis Miesbach":          "landkreis-miesbach",
		"Landkreis Rotenburg (Wümme)": "landkreis-rotenburg-wuemme",
		"Kreis Höxter":                "kreis-hoexter",
		"Saale-Orla-Kreis":            "saale-orla-kreis",
		"Landkreis Bad Dürkheim":      "landkreis-bad-duerkheim",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

// The state is in the class name, which is where Wikidata keeps it. The class without
// one is "former district of Germany" — exactly what must not be imported.
func TestStateOfClass(t *testing.T) {
	cases := map[string]string{
		"Landkreis in Bayern":          "Bayern",
		"Kreis in Nordrhein-Westfalen": "Nordrhein-Westfalen",
		"Landkreis im Saarland":        "Saarland",
		"former district of Germany":   "",
	}
	for in, want := range cases {
		if got := stateOfClass(in); got != want {
			t.Errorf("stateOfClass(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFetchDistrictsReportsAClassThatDidNotAnswer(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		query := r.URL.Query().Get("query")
		switch {
		case contains(query, "P279"):
			_, _ = w.Write([]byte(`{"results":{"bindings":[
				{"class":{"value":"http://www.wikidata.org/entity/Q1"},"classLabel":{"value":"Landkreis in Bayern"}},
				{"class":{"value":"http://www.wikidata.org/entity/Q2"},"classLabel":{"value":"Kreis in Nordrhein-Westfalen"}}
			]}}`))
		case contains(query, "wd:Q1"):
			_, _ = w.Write([]byte(`{"results":{"bindings":[
				{"item":{"value":"http://www.wikidata.org/entity/Q10"},"itemLabel":{"value":"Landkreis Miesbach"},"website":{"value":"https://lk.de/"}}
			]}}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	got, err := New(srv.URL, 5*time.Second).FetchDistricts(context.Background())
	if err == nil {
		t.Fatal("the class that failed went unmentioned")
	}
	// What did answer is kept: a partial import is worth more than none, as long as
	// the gap is named.
	if len(got) != 1 || got[0].State != "Bayern" {
		t.Fatalf("entries = %+v", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
