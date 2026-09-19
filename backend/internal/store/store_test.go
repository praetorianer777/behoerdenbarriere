package store

import (
	"context"
	"os"
	"testing"
	"time"
)

// The migrations are checked against a real Postgres; a mock would miss exactly what
// matters here — that this version accepts the SQL. Without TEST_DATABASE_URL the test
// is skipped.
func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestMigrateIsIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second migration: %v", err)
	}

	var applied int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatalf("schema_migrations: %v", err)
	}
	if applied == 0 {
		t.Fatal("no migration recorded")
	}

	for _, table := range []string{"agencies", "scans", "pages", "violations", "jobs"} {
		var exists bool
		if err := s.Pool.QueryRow(ctx,
			`SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s missing", table)
		}
	}
}

// A second running scan of the same agency would hit the website twice over; the
// partial unique index is the lock against it.
func TestOnlyOneActiveScanPerAgency(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx)

	var agencyID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO agencies (slug, name, url, level) VALUES ($1, $2, $3, 'bund') RETURNING id`,
		"test-"+time.Now().Format("150405.000000"), "Test authority", "https://example.org",
	).Scan(&agencyID); err != nil {
		t.Fatalf("insert agency: %v", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO scans (agency_id, status) VALUES ($1, 'running')`, agencyID); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO scans (agency_id, status) VALUES ($1, 'queued')`, agencyID); err == nil {
		t.Fatal("second active scan was allowed")
	}
}
