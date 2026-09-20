package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
)

// ContactsForScan folds the hosts one scan saw into one row per host and phase. The
// classification does not happen here: the database keeps what was observed, and how we
// read it is decided when it is read.
func (s *Store) ContactsForScan(ctx context.Context, scanID int64) ([]thirdparty.Seen, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT c.host, c.phase::text, sum(c.requests)::int, count(DISTINCT c.page_id)::int
		FROM page_contacts c
		JOIN pages p ON p.id = c.page_id
		WHERE p.scan_id = $1
		GROUP BY c.host, c.phase
		ORDER BY count(DISTINCT c.page_id) DESC, c.host`, scanID)
	if err != nil {
		return nil, fmt.Errorf("contacts of the scan: %w", err)
	}
	return scanSeen(rows)
}

// ContactReach is how far one third party gets across the country: on how many
// authorities it was seen, in which phase.
type ContactReach struct {
	Host     string
	Phase    model.ContactPhase
	Agencies int
	Pages    int
}

// ContactReach counts the authorities per third party, over each authority's newest
// finished scan. Older scans are left out on purpose: a host a city dropped last year
// must not still be counted against it.
func (s *Store) ContactReach(ctx context.Context, limit int) ([]ContactReach, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.Pool.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (s.agency_id) s.id, s.agency_id
			FROM scans s
			JOIN agencies a ON a.id = s.agency_id AND a.active
			WHERE s.status = 'done'
			ORDER BY s.agency_id, s.finished_at DESC
		)
		SELECT c.host, c.phase::text,
		       count(DISTINCT latest.agency_id)::int, count(DISTINCT c.page_id)::int
		FROM latest
		JOIN pages p ON p.scan_id = latest.id
		JOIN page_contacts c ON c.page_id = p.id
		GROUP BY c.host, c.phase
		ORDER BY count(DISTINCT latest.agency_id) DESC, c.host
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("third parties nationwide: %w", err)
	}
	defer rows.Close()

	out := make([]ContactReach, 0)
	for rows.Next() {
		var r ContactReach
		var phase string
		if err := rows.Scan(&r.Host, &phase, &r.Agencies, &r.Pages); err != nil {
			return nil, err
		}
		r.Phase = model.ContactPhase(phase)
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanSeen(rows pgx.Rows) ([]thirdparty.Seen, error) {
	defer rows.Close()

	out := make([]thirdparty.Seen, 0)
	for rows.Next() {
		var seen thirdparty.Seen
		var phase string
		if err := rows.Scan(&seen.Host, &phase, &seen.Requests, &seen.Pages); err != nil {
			return nil, err
		}
		seen.Phase = model.ContactPhase(phase)
		out = append(out, seen)
	}
	return out, rows.Err()
}
