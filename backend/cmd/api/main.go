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

	"github.com/praetorianer777/behoerdenbarriere/internal/api"
	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		return err
	}

	server := api.NewServer(db, cfg.CORSOrigin, cfg.APIKey)
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

	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           server.Routes(),
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
