package store

import (
	"context"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

func TestUpsertAgencyIsIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	slug := "test-" + time.Now().Format("150405.000000")
	a := model.Agency{
		Slug: slug, Name: "Test authority", URL: "https://example.org/",
		Level: model.LevelLand, State: "Hessen", Category: "landesportal",
	}

	first, err := s.UpsertAgency(ctx, a)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	t.Cleanup(func() {
		_, _ = s.Pool.Exec(context.Background(), `DELETE FROM agencies WHERE id = $1`, first)
	})

	// A renamed authority keeps its row — the scan history hangs off it, and a second
	// row would split the history in two.
	a.Name = "Renamed authority"
	a.URL = "https://example.org/neu/"
	second, err := s.UpsertAgency(ctx, a)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if first != second {
		t.Fatalf("a second row was created: %d != %d", first, second)
	}

	var name, url string
	if err := s.Pool.QueryRow(ctx,
		`SELECT name, url FROM agencies WHERE id = $1`, first).Scan(&name, &url); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if name != "Renamed authority" || url != "https://example.org/neu/" {
		t.Fatalf("master data not updated: %s / %s", name, url)
	}
}

// Without a state, a federal authority must not carry an empty string — the filter in
// the ranking would then offer a state with no name.
func TestUpsertAgencyStoresEmptyStateAsNull(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id, err := s.UpsertAgency(ctx, model.Agency{
		Slug: "test-bund-" + time.Now().Format("150405.000000"),
		Name: "Federal authority", URL: "https://example.org/", Level: model.LevelBund,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	t.Cleanup(func() {
		_, _ = s.Pool.Exec(context.Background(), `DELETE FROM agencies WHERE id = $1`, id)
	})

	var isNull bool
	if err := s.Pool.QueryRow(ctx,
		`SELECT state IS NULL FROM agencies WHERE id = $1`, id).Scan(&isNull); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !isNull {
		t.Fatal("empty state was stored as an empty string")
	}
}
