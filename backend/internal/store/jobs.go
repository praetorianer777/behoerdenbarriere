package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
)

// Job is a scan waiting to be run.
type Job struct {
	ID       int64
	AgencyID int64
	Kind     string
	Attempts int
	Agency   model.Agency
}

// After this many attempts a job is put aside for a day. An authority that is down
// should not be retried every five seconds, and it should not block the queue either.
const maxJobAttempts = 5

// EnqueueScan queues an authority. A second call while the job is still waiting
// changes nothing — the unique index on (agency_id, kind) is what makes the queue
// idempotent.
func (s *Store) EnqueueScan(ctx context.Context, agencyID int64) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO jobs (agency_id, kind) VALUES ($1, 'scan')
		ON CONFLICT (agency_id, kind) DO NOTHING`, agencyID)
	return err
}

// EnqueueDue queues every active authority whose last finished scan is older than
// maxAge, and every one that has never been scanned.
func (s *Store) EnqueueDue(ctx context.Context, maxAge time.Duration) (int, error) {
	tag, err := s.Pool.Exec(ctx, `
		INSERT INTO jobs (agency_id, kind)
		SELECT a.id, 'scan'
		FROM agencies a
		LEFT JOIN LATERAL (
			SELECT max(finished_at) AS last_done
			FROM scans
			WHERE agency_id = a.id AND status = 'done'
		) s ON true
		WHERE a.active
		  AND (s.last_done IS NULL OR s.last_done < now() - $1::interval)
		ON CONFLICT (agency_id, kind) DO NOTHING`, maxAge.String())
	if err != nil {
		return 0, fmt.Errorf("queue due authorities: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ClaimJob takes the next due job. SKIP LOCKED lets several workers pull from the
// same queue without two of them scanning the same authority.
func (s *Store) ClaimJob(ctx context.Context, worker string) (*Job, error) {
	var job Job
	err := s.Pool.QueryRow(ctx, `
		UPDATE jobs SET locked_at = now(), locked_by = $1, attempts = attempts + 1
		WHERE id = (
			SELECT j.id FROM jobs j
			JOIN agencies a ON a.id = j.agency_id AND a.active
			WHERE j.locked_at IS NULL AND j.run_after <= now()
			ORDER BY j.run_after
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, agency_id, kind, attempts`, worker,
	).Scan(&job.ID, &job.AgencyID, &job.Kind, &job.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}

	if err := s.Pool.QueryRow(ctx, `
		SELECT id, slug, name, url, level, coalesce(state, ''), coalesce(category, ''), active
		FROM agencies WHERE id = $1`, job.AgencyID,
	).Scan(&job.Agency.ID, &job.Agency.Slug, &job.Agency.Name, &job.Agency.URL,
		&job.Agency.Level, &job.Agency.State, &job.Agency.Category, &job.Agency.Active); err != nil {
		return nil, fmt.Errorf("read authority %d: %w", job.AgencyID, err)
	}
	return &job, nil
}

// FinishJob removes a job that has been done.
func (s *Store) FinishJob(ctx context.Context, jobID int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, jobID)
	return err
}

// RetryJob puts a failed job back with a growing delay.
func (s *Store) RetryJob(ctx context.Context, job *Job, cause error) error {
	backoff := time.Duration(job.Attempts) * 5 * time.Minute
	if job.Attempts >= maxJobAttempts {
		backoff = 24 * time.Hour
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE jobs SET locked_at = NULL, locked_by = NULL,
		                run_after = now() + $2::interval, last_error = $3
		WHERE id = $1`, job.ID, backoff.String(), cause.Error())
	return err
}

// ReleaseStaleJobs frees jobs whose worker died. Without this a crashed process would
// keep its authority locked forever.
func (s *Store) ReleaseStaleJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE jobs SET locked_at = NULL, locked_by = NULL
		WHERE locked_at IS NOT NULL AND locked_at < now() - $1::interval`, olderThan.String())
	if err != nil {
		return 0, err
	}

	// A scan left behind as running belongs to the same crash; while it stands, the
	// partial unique index blocks every new scan of that authority.
	if _, err := s.Pool.Exec(ctx, `
		UPDATE scans SET status = 'failed', finished_at = now(),
		                 error = 'worker did not finish the scan'
		WHERE status = 'running' AND started_at < now() - $1::interval`, olderThan.String()); err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
