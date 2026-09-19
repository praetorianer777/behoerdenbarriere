package store

import (
	"context"
	"fmt"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// UpsertAgency inserts an authority or updates its master data. The slug is the
// identity: renaming an authority or moving its URL must not create a second row,
// because the scan history hangs off the existing one.
func (s *Store) UpsertAgency(ctx context.Context, a model.Agency) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO agencies (slug, name, url, level, state, category)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''))
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			url = EXCLUDED.url,
			level = EXCLUDED.level,
			state = EXCLUDED.state,
			category = EXCLUDED.category,
			updated_at = now()
		RETURNING id`,
		a.Slug, a.Name, a.URL, string(a.Level), a.State, a.Category,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert %s: %w", a.Slug, err)
	}
	return id, nil
}

// CountAgencies returns how many authorities are on record.
func (s *Store) CountAgencies(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM agencies`).Scan(&n)
	return n, err
}
