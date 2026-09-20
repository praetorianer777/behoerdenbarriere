package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/storetest"
)

func freshAgency(t *testing.T, s *store.Store) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := s.UpsertAgency(ctx, model.Agency{
		Slug:  "test-" + time.Now().Format("150405.000000000"),
		Name:  "Test authority",
		URL:   "https://example.org/",
		Level: model.LevelBund,
	})
	if err != nil {
		t.Fatalf("create authority: %v", err)
	}
	t.Cleanup(func() {
		_, _ = s.Pool.Exec(context.Background(), `DELETE FROM agencies WHERE id = $1`, id)
	})
	return id
}

func TestEnqueueScanIsIdempotent(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)

	for range 3 {
		if err := s.EnqueueScan(ctx, id); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	var queued int
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM jobs WHERE agency_id = $1`, id).Scan(&queued); err != nil {
		t.Fatalf("count: %v", err)
	}
	if queued != 1 {
		t.Fatalf("%d jobs queued, want 1", queued)
	}
}

func TestClaimJobCarriesTheAgency(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)
	if err := s.EnqueueScan(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	job := claimOurs(t, s, id)
	if job == nil {
		t.Fatal("no job claimed")
	}
	if job.Agency.ID != id || job.Agency.URL != "https://example.org/" {
		t.Fatalf("authority not carried along: %+v", job.Agency)
	}
	if job.Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", job.Attempts)
	}
}

// Two workers pulling at once must not get the same authority — the website would be
// scanned twice in parallel.
func TestClaimJobHandsOutAJobOnlyOnce(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)
	if err := s.EnqueueScan(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	first := claimOurs(t, s, id)
	if first == nil {
		t.Fatal("first claim empty")
	}
	if second := claimOurs(t, s, id); second != nil {
		t.Fatalf("the job was handed out a second time: %+v", second)
	}
}

func TestRetryJobDelaysAndKeepsTheCause(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)
	if err := s.EnqueueScan(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	job := claimOurs(t, s, id)

	if err := s.RetryJob(ctx, job, errors.New("chrome unreachable")); err != nil {
		t.Fatalf("retry: %v", err)
	}

	var runAfter time.Time
	var lastError string
	var locked *time.Time
	if err := s.Pool.QueryRow(ctx,
		`SELECT run_after, coalesce(last_error, ''), locked_at FROM jobs WHERE id = $1`, job.ID,
	).Scan(&runAfter, &lastError, &locked); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if locked != nil {
		t.Error("job still locked")
	}
	if !runAfter.After(time.Now().Add(time.Minute)) {
		t.Errorf("run_after = %v — no delay", runAfter)
	}
	if lastError != "chrome unreachable" {
		t.Errorf("cause = %q", lastError)
	}

	// While it is waiting, it must not be handed out again.
	if again := claimOurs(t, s, id); again != nil {
		t.Fatal("a delayed job was claimed")
	}
}

func TestFinishJobRemovesIt(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)
	if err := s.EnqueueScan(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	job := claimOurs(t, s, id)

	if err := s.FinishJob(ctx, job.ID); err != nil {
		t.Fatalf("finish: %v", err)
	}
	var left int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE id = $1`, job.ID).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 0 {
		t.Fatal("job still in the queue")
	}
}

// A worker that dies must not keep its authority locked forever.
func TestReleaseStaleJobs(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)
	if err := s.EnqueueScan(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	job := claimOurs(t, s, id)

	if _, err := s.Pool.Exec(ctx,
		`UPDATE jobs SET locked_at = now() - interval '2 hours' WHERE id = $1`, job.ID); err != nil {
		t.Fatalf("age the lock: %v", err)
	}
	if _, err := s.ReleaseStaleJobs(ctx, time.Hour); err != nil {
		t.Fatalf("release: %v", err)
	}
	if again := claimOurs(t, s, id); again == nil {
		t.Fatal("job stayed locked")
	}
}

// A scan left behind as running blocks every later scan of that authority through the
// partial unique index, so releasing has to close it too.
func TestReleaseStaleJobsClosesAbandonedScans(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)

	var scanID int64
	if err := s.Pool.QueryRow(ctx,
		`INSERT INTO scans (agency_id, status, started_at) VALUES ($1, 'running', now() - interval '2 hours') RETURNING id`,
		id).Scan(&scanID); err != nil {
		t.Fatalf("create scan: %v", err)
	}

	if _, err := s.ReleaseStaleJobs(ctx, time.Hour); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := s.StartScan(ctx, id, nil); err != nil {
		t.Fatalf("no new scan possible: %v", err)
	}
}

func TestEnqueueDueSkipsFreshlyScannedAuthorities(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)

	if _, err := s.Pool.Exec(ctx,
		`INSERT INTO scans (agency_id, status, finished_at, score, grade)
		 VALUES ($1, 'done', now(), 88, 'B')`, id); err != nil {
		t.Fatalf("create scan: %v", err)
	}

	if _, err := s.EnqueueDue(ctx, 7*24*time.Hour); err != nil {
		t.Fatalf("enqueue due: %v", err)
	}
	var queued int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE agency_id = $1`, id).Scan(&queued); err != nil {
		t.Fatalf("count: %v", err)
	}
	if queued != 0 {
		t.Fatal("a freshly scanned authority was queued again")
	}

	// Once the scan is old enough, it is due again.
	if _, err := s.Pool.Exec(ctx,
		`UPDATE scans SET finished_at = now() - interval '30 days' WHERE agency_id = $1`, id); err != nil {
		t.Fatalf("age the scan: %v", err)
	}
	if _, err := s.EnqueueDue(ctx, 7*24*time.Hour); err != nil {
		t.Fatalf("enqueue due: %v", err)
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE agency_id = $1`, id).Scan(&queued); err != nil {
		t.Fatalf("count: %v", err)
	}
	if queued != 1 {
		t.Fatal("an authority due for a rescan was not queued")
	}
}

func TestEnqueueDueSkipsInactiveAuthorities(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()
	id := freshAgency(t, s)
	if _, err := s.Pool.Exec(ctx, `UPDATE agencies SET active = false WHERE id = $1`, id); err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	if _, err := s.EnqueueDue(ctx, time.Hour); err != nil {
		t.Fatalf("enqueue due: %v", err)
	}
	var queued int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE agency_id = $1`, id).Scan(&queued); err != nil {
		t.Fatalf("count: %v", err)
	}
	if queued != 0 {
		t.Fatal("an authority taken out of the ranking was queued")
	}
}

func TestQueueStatsCountsWaitingAndRunningJobs(t *testing.T) {
	s := storetest.New(t)
	ctx := context.Background()

	empty, err := s.QueueStats(ctx)
	if err != nil {
		t.Fatalf("queue stats: %v", err)
	}
	if empty.Queued != 0 || empty.Running != 0 || empty.OldestAge != 0 {
		t.Fatalf("empty queue = %+v", empty)
	}

	if err := s.EnqueueScan(ctx, freshAgency(t, s)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if job, err := s.ClaimJob(ctx, "test"); err != nil || job == nil {
		t.Fatalf("claim: %v %v", job, err)
	}

	if err := s.EnqueueScan(ctx, freshAgency(t, s)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := s.Pool.Exec(ctx,
		`UPDATE jobs SET run_after = now() - interval '2 minutes' WHERE locked_at IS NULL`); err != nil {
		t.Fatalf("age the job: %v", err)
	}

	stats, err := s.QueueStats(ctx)
	if err != nil {
		t.Fatalf("queue stats: %v", err)
	}
	if stats.Queued != 1 || stats.Running != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	if stats.OldestAge < time.Minute {
		t.Fatalf("oldest job is %v old, want about two minutes", stats.OldestAge)
	}
}

// Other tests run in the same database, so a claim only counts when it is ours.
func claimOurs(t *testing.T, s *store.Store, agencyID int64) *store.Job {
	t.Helper()
	for range 50 {
		job, err := s.ClaimJob(context.Background(), "test")
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
		if job == nil {
			return nil
		}
		if job.AgencyID == agencyID {
			return job
		}
	}
	t.Fatal("too many foreign jobs in the queue")
	return nil
}
