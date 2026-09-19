package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
)

// ErrScanInFlight means a scan of this authority is already queued or running.
var ErrScanInFlight = errors.New("a scan of this authority is already running")

// StartScan opens a scan. The partial unique index in the schema is what enforces
// one at a time; catching its violation here turns a race between the scheduler and a
// manual rescan into a plain answer instead of an error nobody can read.
func (s *Store) StartScan(ctx context.Context, agencyID int64, cfg map[string]any) (int64, error) {
	if cfg == nil {
		cfg = map[string]any{}
	}
	var id int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO scans (agency_id, status, config)
		VALUES ($1, 'running', $2)
		RETURNING id`, agencyID, cfg).Scan(&id)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return 0, ErrScanInFlight
	}
	if err != nil {
		return 0, fmt.Errorf("start scan: %w", err)
	}
	return id, nil
}

// FinishScan writes the pages, their violations and the computed score in one
// transaction. Half a scan in the database would look like a complete one in the
// ranking, so either all of it lands or none of it.
func (s *Store) FinishScan(ctx context.Context, scanID int64, pages []model.PageResult, result scoring.Result) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var failed int
	for _, page := range pages {
		if page.Failed() {
			failed++
		}
		var pageID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO pages (scan_id, url, depth, is_entry, priority, title,
			                   http_status, dom_nodes, page_score, load_ms, error)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, 0), $8, $9, NULLIF($10, 0), NULLIF($11, ''))
			ON CONFLICT (scan_id, url) DO UPDATE SET url = EXCLUDED.url
			RETURNING id`,
			scanID, page.URL, page.Depth, page.IsEntry, page.Priority, page.Title,
			page.HTTPStatus, page.DOMNodes, scoring.PageScore(page), page.LoadMS, page.Err,
		).Scan(&pageID); err != nil {
			return fmt.Errorf("save page %s: %w", page.URL, err)
		}

		for _, v := range page.Violations {
			if _, err := tx.Exec(ctx, `
				INSERT INTO violations (page_id, rule_id, impact, description, help, help_url,
				                        wcag_tags, principle, node_count, sample_html, sample_target)
				VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
				        $7, $8, $9, NULLIF($10, ''), NULLIF($11, ''))`,
				pageID, v.RuleID, string(v.Impact), v.Description, v.Help, v.HelpURL,
				v.WCAGTags, string(v.Principle), v.NodeCount, v.SampleHTML, v.SampleTarget,
			); err != nil {
				return fmt.Errorf("save violation %s: %w", v.RuleID, err)
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE scans SET
			status = 'done', finished_at = now(),
			score = $2, grade = $3,
			score_perceivable = $4, score_operable = $5,
			score_understandable = $6, score_robust = $7,
			pages_scanned = $8, pages_failed = $9
		WHERE id = $1`,
		scanID, result.Score, nullIfEmpty(result.Grade),
		result.Principles[model.Perceivable], result.Principles[model.Operable],
		result.Principles[model.Understandable], result.Principles[model.Robust],
		result.Pages, failed,
	); err != nil {
		return fmt.Errorf("close scan: %w", err)
	}
	return tx.Commit(ctx)
}

// FailScan records why a scan produced nothing.
func (s *Store) FailScan(ctx context.Context, scanID int64, cause error) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE scans SET status = 'failed', finished_at = now(), error = $2
		WHERE id = $1`, scanID, cause.Error())
	return err
}

// LatestScore returns the most recent finished score of an authority.
func (s *Store) LatestScore(ctx context.Context, agencyID int64) (float64, string, time.Time, error) {
	var score float64
	var grade string
	var at time.Time
	err := s.Pool.QueryRow(ctx, `
		SELECT coalesce(score, 0), coalesce(grade, ''), finished_at
		FROM scans
		WHERE agency_id = $1 AND status = 'done'
		ORDER BY finished_at DESC
		LIMIT 1`, agencyID).Scan(&score, &grade, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", time.Time{}, nil
	}
	return score, grade, at, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
