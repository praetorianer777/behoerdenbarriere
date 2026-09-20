// Command worker runs the scans: it takes jobs off the queue, has the authority
// crawled and checked, and puts due authorities back on the queue.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/exaring/otelpgx"

	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/crawler"
	"github.com/praetorianer777/behoerdenbarriere/internal/lighthouse"
	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/pipeline"
	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/telemetry"
)

// A worker that dies leaves its job locked. After this long another worker may take
// it — long enough that a running scan is not taken away from a live worker.
const staleAfter = 30 * time.Minute

func main() {
	if err := run(); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(telemetry.NewLogger(os.Stderr, cfg.LogLevel))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Config{
		ServiceName: cfg.OTel.ServiceNameOr("behoerdenbarriere-worker"),
		Endpoint:    cfg.OTel.Endpoint,
		Protocol:    cfg.OTel.Protocol,
		SampleRatio: cfg.OTel.SampleRatio,
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := shutdownTelemetry(context.Background()); err != nil {
			slog.Warn("telemetry shutdown", "error", err)
		}
	}()

	var dbOpts []store.Option
	if cfg.OTel.Enabled() {
		dbOpts = append(dbOpts, store.WithQueryTracer(otelpgx.NewTracer()))
	}
	db, err := store.Open(ctx, cfg.DatabaseURL, dbOpts...)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		return err
	}

	if cfg.OTel.Enabled() {
		if err := otelpgx.RecordStats(db.Pool); err != nil {
			return err
		}
		if err := telemetry.ObserveQueue(func(ctx context.Context) (telemetry.QueueStats, error) {
			stats, err := db.QueueStats(ctx)
			return telemetry.QueueStats(stats), err
		}); err != nil {
			return err
		}
	}

	sc, err := scanner.New(ctx, scanner.Options{
		ChromeURL:   cfg.ChromeURL,
		PageTimeout: cfg.Scan.PageTimeout,
	})
	if err != nil {
		return err
	}
	defer sc.Close()

	crawlCfg := crawler.Config{
		MaxPages:   cfg.Crawl.MaxPages,
		MaxDepth:   cfg.Crawl.MaxDepth,
		RatePerSec: cfg.Crawl.RatePerSec,
		Timeout:    cfg.Crawl.Timeout,
	}
	pipe := pipeline.New(db, crawler.New(telemetry.TracePageScanner(sc), crawlCfg), crawlCfg, slog.Default()).
		WithAuditor(lighthouse.New(cfg.LighthouseURL, cfg.LighthouseTimeout)).
		WithMail(maildns.New(nil, 15*time.Second))

	name, _ := os.Hostname()
	slog.Info("worker ready", "name", name, "chrome", cfg.ChromeURL)

	ticker := time.NewTicker(cfg.Worker.PollInterval)
	defer ticker.Stop()

	for {
		// Work until the queue is empty, and only then wait — otherwise a hundred
		// authorities would be scanned at one per poll interval.
		for {
			done, err := step(ctx, db, pipe, name, cfg.Worker.RescanInterval)
			if err != nil {
				slog.Error("job failed", "error", err)
			}
			if !done || ctx.Err() != nil {
				break
			}
		}

		select {
		case <-ctx.Done():
			slog.Info("worker stopped")
			return nil
		case <-ticker.C:
		}
	}
}

// step does one job and reports whether there was one.
func step(ctx context.Context, db *store.Store, pipe *pipeline.Pipeline, worker string, rescan time.Duration) (bool, error) {
	if _, err := db.ReleaseStaleJobs(ctx, staleAfter); err != nil {
		return false, err
	}
	if n, err := db.EnqueueDue(ctx, rescan); err != nil {
		return false, err
	} else if n > 0 {
		slog.Info("authorities queued", "count", n)
	}

	job, err := db.ClaimJob(ctx, worker)
	if err != nil || job == nil {
		return false, err
	}

	if _, err := pipe.Run(ctx, job.Agency); err != nil {
		if retryErr := db.RetryJob(ctx, job, err); retryErr != nil {
			return true, retryErr
		}
		return true, err
	}
	return true, db.FinishJob(ctx, job.ID)
}
