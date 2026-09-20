package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// UpsertAgency inserts an authority or updates its master data. The slug is the
// identity: renaming an authority or moving its URL must not create a second row,
// because the scan history hangs off the existing one.
func (s *Store) UpsertAgency(ctx context.Context, a model.Agency) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO agencies (slug, name, url, level, state, category, source)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), 'seed')
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

// UpsertImported adds an authority that came from an import.
//
// It never overwrites an entry from the hand-kept list: that one has been checked, the
// import has not. Where both know the same authority, the checked entry stands and the
// import only fills what is missing.
func (s *Store) UpsertImported(ctx context.Context, a model.Agency) (int64, bool, error) {
	if a.ExternalID == "" {
		return 0, false, fmt.Errorf("imported entry %s without an identifier at its source", a.Slug)
	}

	// Known by its identifier at the source: the import owns this row and may update it.
	var id int64
	err := s.Pool.QueryRow(ctx, `
		UPDATE agencies SET name = $3, url = $4, level = $5, state = NULLIF($6, ''),
		                    category = NULLIF($7, ''), updated_at = now()
		WHERE source = $1 AND external_id = $2
		RETURNING id`,
		string(a.Source), a.ExternalID, a.Name, a.URL, string(a.Level), a.State, a.Category,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, fmt.Errorf("update imported %s: %w", a.Slug, err)
	}

	// Known by slug or by host: somebody is already covering this authority, and if
	// that entry was kept by hand it stays as it is.
	if existing, err := s.findAgency(ctx, a); err != nil {
		return 0, false, err
	} else if existing != 0 {
		return existing, false, nil
	}

	err = s.Pool.QueryRow(ctx, `
		INSERT INTO agencies (slug, name, url, level, state, category, source, external_id)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8)
		RETURNING id`,
		a.Slug, a.Name, a.URL, string(a.Level), a.State, a.Category,
		string(a.Source), a.ExternalID,
	).Scan(&id)
	if err != nil {
		return 0, false, fmt.Errorf("insert imported %s: %w", a.Slug, err)
	}
	return id, true, nil
}

// findAgency looks for an entry that already covers this authority — by its slug, or
// by the host of its website, because the same authority reached under two slugs would
// appear twice in the ranking.
func (s *Store) findAgency(ctx context.Context, a model.Agency) (int64, error) {
	host := hostOf(a.URL)
	var id int64
	err := s.Pool.QueryRow(ctx, `
		SELECT id FROM agencies
		WHERE slug = $1
		   OR ($2 <> '' AND regexp_replace(split_part(split_part(url, '//', 2), '/', 1), '^www\.', '') = $2)
		LIMIT 1`, a.Slug, host).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("look for %s: %w", a.Slug, err)
	}
	return id, nil
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}
