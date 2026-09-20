package telemetry

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Scan outcomes as they are reported on the status attribute.
const (
	StatusDone   = "done"
	StatusFailed = "failed"
)

type scanInstruments struct {
	scans    metric.Int64Counter
	duration metric.Float64Histogram
	pages    metric.Int64Counter
}

// The instruments are built on first use rather than in Setup, so that a package can
// report without being handed anything, and so that the no-op provider is used when
// telemetry is off.
var scanMetrics = sync.OnceValue(func() *scanInstruments {
	m := otel.GetMeterProvider().Meter(scopeName)

	scans, err1 := m.Int64Counter("scans.total",
		metric.WithDescription("Finished scans by outcome"))
	duration, err2 := m.Float64Histogram("scan.duration",
		metric.WithDescription("Duration of one authority scan"), metric.WithUnit("s"))
	pages, err3 := m.Int64Counter("scan.pages",
		metric.WithDescription("Pages checked"), metric.WithUnit("{page}"))

	if err := errors.Join(err1, err2, err3); err != nil {
		otel.Handle(err)
		return nil
	}
	return &scanInstruments{scans: scans, duration: duration, pages: pages}
})

// RecordScan notes one finished scan. Pages and duration together are what says
// whether a slow scan was a big site or a slow one.
func RecordScan(ctx context.Context, status string, pages int, d time.Duration) {
	m := scanMetrics()
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(attribute.String("status", status))
	m.scans.Add(ctx, 1, attrs)
	m.duration.Record(ctx, d.Seconds(), attrs)
	if pages > 0 {
		m.pages.Add(ctx, int64(pages), attrs)
	}
}

// QueueStats is what the worker reports about the job queue.
type QueueStats struct {
	Queued    int
	Running   int
	OldestAge time.Duration
}

// ObserveQueue has the queue read out whenever metrics are collected. With telemetry
// off the callback is never called, so nothing is asked of the database.
func ObserveQueue(read func(context.Context) (QueueStats, error)) error {
	m := otel.GetMeterProvider().Meter(scopeName)

	queued, err := m.Int64ObservableGauge("jobs.queued",
		metric.WithDescription("Jobs waiting in the queue"), metric.WithUnit("{job}"))
	if err != nil {
		return err
	}
	running, err := m.Int64ObservableGauge("jobs.running",
		metric.WithDescription("Jobs a worker has taken on"), metric.WithUnit("{job}"))
	if err != nil {
		return err
	}
	// The age of the oldest waiting job is the one number that shows the workers are
	// not keeping up, no matter how long the queue is.
	oldest, err := m.Float64ObservableGauge("jobs.oldest_age",
		metric.WithDescription("Age of the oldest waiting job"), metric.WithUnit("s"))
	if err != nil {
		return err
	}

	_, err = m.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		stats, err := read(ctx)
		if err != nil {
			return err
		}
		o.ObserveInt64(queued, int64(stats.Queued))
		o.ObserveInt64(running, int64(stats.Running))
		o.ObserveFloat64(oldest, stats.OldestAge.Seconds())
		return nil
	}, queued, running, oldest)
	return err
}
