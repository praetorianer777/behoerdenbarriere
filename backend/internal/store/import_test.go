package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

func imported(slug, name, url, id string) model.Agency {
	return model.Agency{
		Slug: slug, Name: name, URL: url, Level: model.LevelKreis,
		State: "Bayern", Source: model.SourceWikidata, ExternalID: id,
	}
}

func TestUpsertImportedAddsAndUpdatesItsOwn(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	id, isNew, err := s.UpsertImported(ctx, imported("landkreis-miesbach",
		"Landkreis Miesbach", "https://www.landkreis-miesbach.de/", "Q10523"))
	if err != nil || !isNew {
		t.Fatalf("first import: id=%d new=%v err=%v", id, isNew, err)
	}

	// Recognised by its identifier at the source, even after a rename.
	again, isNew, err := s.UpsertImported(ctx, imported("landkreis-miesbach-neu",
		"Landkreis Miesbach (neu)", "https://www.lk-miesbach.de/", "Q10523"))
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if isNew || again != id {
		t.Fatalf("a second row was created: %d vs %d", again, id)
	}

	var name, url string
	if err := s.Pool.QueryRow(ctx,
		`SELECT name, url FROM agencies WHERE id = $1`, id).Scan(&name, &url); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if name != "Landkreis Miesbach (neu)" || url != "https://www.lk-miesbach.de/" {
		t.Fatalf("the import did not update its own row: %s / %s", name, url)
	}
}

// The hand-kept list has been checked, the import has not. Where both know the same
// authority, the checked entry stands — otherwise a careless Wikidata edit would move
// a scan to a different website.
func TestUpsertImportedLeavesHandKeptEntriesAlone(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	slug := "stadt-muenchen-" + time.Now().Format("150405.000")
	kept, err := s.UpsertAgency(ctx, model.Agency{
		Slug: slug, Name: "München", URL: "https://stadt.muenchen.de/",
		Level: model.LevelKommune, State: "Bayern",
	})
	if err != nil {
		t.Fatalf("hand-kept entry: %v", err)
	}

	got, isNew, err := s.UpsertImported(ctx, model.Agency{
		Slug: slug, Name: "Landeshauptstadt München", URL: "https://www.muenchen.de/",
		Level: model.LevelKreis, State: "Bayern",
		Source: model.SourceWikidata, ExternalID: "Q1726",
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if isNew || got != kept {
		t.Fatalf("the import created its own row: %d vs %d", got, kept)
	}

	var name, url, source string
	if err := s.Pool.QueryRow(ctx,
		`SELECT name, url, source::text FROM agencies WHERE id = $1`, kept).Scan(&name, &url, &source); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if name != "München" || url != "https://stadt.muenchen.de/" || source != "seed" {
		t.Fatalf("the checked entry was overwritten: %s / %s / %s", name, url, source)
	}
}

// The same authority under two slugs would stand twice in the ranking, so the website
// decides as well.
func TestUpsertImportedRecognisesTheSameHost(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	slug := "kreis-steinfurt-" + time.Now().Format("150405.000")
	kept, err := s.UpsertAgency(ctx, model.Agency{
		Slug: slug, Name: "Kreis Steinfurt", URL: "https://www.kreis-steinfurt.de/",
		Level: model.LevelKreis, State: "Nordrhein-Westfalen",
	})
	if err != nil {
		t.Fatalf("hand-kept entry: %v", err)
	}

	got, isNew, err := s.UpsertImported(ctx, model.Agency{
		Slug: "kreis-steinfurt-wikidata", Name: "Kreis Steinfurt",
		URL: "https://kreis-steinfurt.de/", Level: model.LevelKreis,
		State: "Nordrhein-Westfalen", Source: model.SourceWikidata, ExternalID: "Q6187",
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if isNew || got != kept {
		t.Fatalf("the same authority was added twice: %d vs %d", got, kept)
	}
}

// An imported entry without an identifier at its source could never be recognised
// again, and a second import would pile up copies.
func TestUpsertImportedNeedsAnIdentifier(t *testing.T) {
	s := storetest.New(t)

	if _, _, err := s.UpsertImported(context.Background(), model.Agency{
		Slug: "ohne-kennung", Name: "Ohne Kennung", URL: "https://a.de/",
		Level: model.LevelKreis, Source: model.SourceWikidata,
	}); err == nil {
		t.Fatal("an entry without an identifier was accepted")
	}
}
