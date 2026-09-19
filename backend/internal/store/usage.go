package store

import (
	"context"
	"fmt"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/usage"
)

// UsageDay is one day of the counter: how many distinct visitors were seen and how
// many pages they looked at.
type UsageDay struct {
	Day      time.Time
	Visitors int64
	Views    int64
}

// UsageKey is one counted thing — a page, an authority, an endpoint — with its total
// over the window.
type UsageKey struct {
	Key   string
	Count int64
}

// UsageSummary is what the statistics page shows.
type UsageSummary struct {
	Since     time.Time
	Until     time.Time
	Visitors  int64
	Views     int64
	Days      []UsageDay
	Pages     []UsageKey
	Agencies  []UsageKey
	Endpoints []UsageKey
}

const maxUsageAgencies = 20

// AddUsage adds a batch of counters. The upsert is what keeps the table at one row
// per day, kind and key no matter how often the batch arrives.
func (s *Store) AddUsage(ctx context.Context, counts []usage.Count) error {
	if len(counts) == 0 {
		return nil
	}
	days := make([]time.Time, len(counts))
	kinds := make([]string, len(counts))
	keys := make([]string, len(counts))
	values := make([]int64, len(counts))
	for i, c := range counts {
		days[i], kinds[i], keys[i], values[i] = c.Day, string(c.Kind), c.Key, c.Count
	}

	_, err := s.Pool.Exec(ctx, `
		INSERT INTO usage_counters (day, kind, key, count)
		SELECT * FROM unnest($1::date[], $2::text[], $3::text[], $4::bigint[])
		ON CONFLICT (day, kind, key)
		DO UPDATE SET count = usage_counters.count + EXCLUDED.count`,
		days, kinds, keys, values)
	if err != nil {
		return fmt.Errorf("add usage counters: %w", err)
	}
	return nil
}

// DropVisitorHashes removes the visitor rows of older days. Their counts are already
// in the daily totals, and a hash that nothing is computed from any more is only a
// leftover.
func (s *Store) DropVisitorHashes(ctx context.Context, before time.Time) error {
	_, err := s.Pool.Exec(ctx,
		`DELETE FROM usage_counters WHERE kind = 'visitor' AND day < $1::date`, before)
	if err != nil {
		return fmt.Errorf("drop visitor hashes: %w", err)
	}
	return nil
}

// Usage sums the counters of the last days. Visitors are counted per day and then
// added up: the same person on two days is two visitors, because the daily salt makes
// them impossible to recognise — the count is an upper bound and says so on the page.
func (s *Store) Usage(ctx context.Context, days int) (*UsageSummary, error) {
	if days < 1 {
		days = 30
	}
	summary := &UsageSummary{Days: []UsageDay{}, Pages: []UsageKey{},
		Agencies: []UsageKey{}, Endpoints: []UsageKey{}}

	if err := s.Pool.QueryRow(ctx,
		`SELECT current_date - ($1::int - 1), current_date`, days,
	).Scan(&summary.Since, &summary.Until); err != nil {
		return nil, fmt.Errorf("usage window: %w", err)
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT day,
		       count(*) FILTER (WHERE kind = 'visitor') AS visitors,
		       coalesce(sum(count) FILTER (WHERE kind = 'page'), 0) AS views
		FROM usage_counters
		WHERE day >= $1
		GROUP BY day
		ORDER BY day`, summary.Since)
	if err != nil {
		return nil, fmt.Errorf("usage per day: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d UsageDay
		if err := rows.Scan(&d.Day, &d.Visitors, &d.Views); err != nil {
			return nil, err
		}
		summary.Visitors += d.Visitors
		summary.Views += d.Views
		summary.Days = append(summary.Days, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	keyRows, err := s.Pool.Query(ctx, `
		SELECT kind, key, sum(count)
		FROM usage_counters
		WHERE day >= $1 AND kind <> 'visitor'
		GROUP BY kind, key
		ORDER BY sum(count) DESC, key`, summary.Since)
	if err != nil {
		return nil, fmt.Errorf("usage per key: %w", err)
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var kind string
		var entry UsageKey
		if err := keyRows.Scan(&kind, &entry.Key, &entry.Count); err != nil {
			return nil, err
		}
		switch usage.Kind(kind) {
		case usage.KindPage:
			summary.Pages = append(summary.Pages, entry)
		case usage.KindAgency:
			if len(summary.Agencies) < maxUsageAgencies {
				summary.Agencies = append(summary.Agencies, entry)
			}
		case usage.KindAPI:
			summary.Endpoints = append(summary.Endpoints, entry)
		}
	}
	if err := keyRows.Err(); err != nil {
		return nil, err
	}
	return summary, nil
}
