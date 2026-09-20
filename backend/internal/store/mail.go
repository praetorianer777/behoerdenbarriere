package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
)

// SaveMail stores what a domain publishes about its mail. One row per authority: this
// is a property of the domain, not of a single scan, and a history of it would say
// little that the current state does not.
func (s *Store) SaveMail(ctx context.Context, agencyID int64, record maildns.Record) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO agency_mail (agency_id, checked_at, provider, record, error)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		ON CONFLICT (agency_id) DO UPDATE SET
			checked_at = EXCLUDED.checked_at, provider = EXCLUDED.provider,
			record = EXCLUDED.record, error = EXCLUDED.error`,
		agencyID, record.CheckedAt, string(record.Provider), record, record.Err)
	if err != nil {
		return fmt.Errorf("save mail record: %w", err)
	}
	return nil
}

// MailForAgency returns what was last read for one authority.
func (s *Store) MailForAgency(ctx context.Context, agencyID int64) (*maildns.Record, error) {
	var record maildns.Record
	err := s.Pool.QueryRow(ctx,
		`SELECT record FROM agency_mail WHERE agency_id = $1`, agencyID).Scan(&record)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mail record: %w", err)
	}
	return &record, nil
}

// MailCount is one group of authorities: how many of them, with which provider.
type MailCount struct {
	Name     string
	Provider string
	Agencies int
}

// MailSummary is the nationwide picture of where authority mail is received.
type MailSummary struct {
	Total      int
	ByProvider []MailCount
	ByState    []MailCount
	ByLevel    []MailCount
	CheckedAt  *time.Time
}

// MailOverview counts the authorities per provider, and per state and level. Only
// active authorities count: a record for one we no longer check would age silently.
func (s *Store) MailOverview(ctx context.Context) (*MailSummary, error) {
	summary := &MailSummary{}

	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*), max(m.checked_at)
		FROM agency_mail m JOIN agencies a ON a.id = m.agency_id AND a.active`,
	).Scan(&summary.Total, &summary.CheckedAt); err != nil {
		return nil, fmt.Errorf("mail overview: %w", err)
	}

	groups := []struct {
		column string
		into   *[]MailCount
	}{
		{"''", &summary.ByProvider},
		{"coalesce(a.state, '')", &summary.ByState},
		{"a.level::text", &summary.ByLevel},
	}
	for _, group := range groups {
		rows, err := s.Pool.Query(ctx, `
			SELECT `+group.column+` AS name, m.provider, count(*)::int
			FROM agency_mail m JOIN agencies a ON a.id = m.agency_id AND a.active
			GROUP BY 1, 2
			ORDER BY count(*) DESC, 1`)
		if err != nil {
			return nil, fmt.Errorf("mail overview: %w", err)
		}
		counts, err := scanMailCounts(rows)
		if err != nil {
			return nil, err
		}
		*group.into = counts
	}
	return summary, nil
}

func scanMailCounts(rows pgx.Rows) ([]MailCount, error) {
	defer rows.Close()

	out := make([]MailCount, 0)
	for rows.Next() {
		var count MailCount
		if err := rows.Scan(&count.Name, &count.Provider, &count.Agencies); err != nil {
			return nil, err
		}
		out = append(out, count)
	}
	return out, rows.Err()
}

// AgenciesToResolve lists the active authorities with their URL, so the DNS records can
// be refreshed for all of them.
func (s *Store) AgenciesToResolve(ctx context.Context) ([]MailTarget, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT a.id, a.slug, a.url FROM agencies a WHERE a.active ORDER BY a.id`)
	if err != nil {
		return nil, fmt.Errorf("authorities for the DNS lookup: %w", err)
	}
	defer rows.Close()

	out := make([]MailTarget, 0)
	for rows.Next() {
		var target MailTarget
		if err := rows.Scan(&target.ID, &target.Slug, &target.URL); err != nil {
			return nil, err
		}
		out = append(out, target)
	}
	return out, rows.Err()
}

// MailTarget is one authority to look up.
type MailTarget struct {
	ID   int64
	Slug string
	URL  string
}
