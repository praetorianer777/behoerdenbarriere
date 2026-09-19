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

	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           telemetry.WrapHandler(api.NewServer(db, cfg.CORSOrigin, cfg.APIKey).Routes(), service),
		ReadHeaderTimeout: 10 * time.Second,
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
