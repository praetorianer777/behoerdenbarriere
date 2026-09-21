package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/exaring/otelpgx"

	"github.com/praetorianer777/behoerdenbarriere/internal/api"
	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/telemetry"
	"github.com/praetorianer777/behoerdenbarriere/internal/usage"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
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

	service := cfg.OTel.ServiceNameOr("behoerdenbarriere-api")
	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Config{
		ServiceName: service,
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

	if cfg.OTel.Enabled() {
		if err := otelpgx.RecordStats(db.Pool); err != nil {
			return err
		}
	}

	if err := db.Migrate(ctx); err != nil {
		return err
	}

	server := api.NewServer(db, api.Options{
		CORSOrigin: cfg.CORSOrigin,
		APIKeys:    cfg.APIKeys(),
		Operator: api.Operator{
			Name: cfg.Operator.Name, Street: cfg.Operator.Street, City: cfg.Operator.City,
			Country: cfg.Operator.Country, Email: cfg.Operator.Email, Phone: cfg.Operator.Phone,
			VATID: cfg.Operator.VATID, Hosting: cfg.Operator.Hosting,
		},
		Limits: api.Limits{
			ReadPerMinute:      cfg.API.ReadPerMinute,
			ReadBurst:          cfg.API.ReadBurst,
			ExpensivePerMinute: cfg.API.ExpensivePerMinute,
			ExpensiveBurst:     cfg.API.ExpensiveBurst,
			KeyPerMinute:       cfg.API.KeyPerMinute,
			KeyBurst:           cfg.API.KeyBurst,
			TrustedProxies:     cfg.API.TrustedProxies,
			RescanPerAgency:    cfg.API.RescanPerAgency,
			MaxBodyBytes:       cfg.API.MaxBodyBytes,
			HistoryPoints:      cfg.API.HistoryPoints,
			ListItems:          cfg.API.ListItems,
			CacheMaxAge:        cfg.API.CacheMaxAge,
			BucketIdleTTL:      cfg.API.BucketIdleTTL,
			RequestTimeout:     cfg.API.RequestTimeout,
		},
	})

	if cfg.Usage.Enabled {
		recorder := usage.NewRecorder(db, cfg.Usage.RetainDays)
		server.WithUsage(recorder)
		done := make(chan struct{})
		go func() {
			defer close(done)
			recorder.Run(ctx, cfg.Usage.FlushInterval)
		}()
		// The last counts are only in memory; waiting for the flush is what keeps
		// them.
		defer func() { <-done }()
	}

	// Timeouts on every stage: a client that opens a connection and then falls silent
	// must not hold a slot for good.
	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           telemetry.WrapHandler(server.Routes(), service),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.API.ReadTimeout,
		WriteTimeout:      cfg.API.WriteTimeout,
		IdleTimeout:       cfg.API.IdleTimeout,
		MaxHeaderBytes:    16 << 10,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", cfg.APIAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
