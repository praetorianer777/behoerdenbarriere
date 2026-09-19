package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
)

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	slog.Info("worker ready", "chrome", cfg.ChromeURL, "poll", cfg.Worker.PollInterval)
	<-ctx.Done()
	slog.Info("worker stopped")
	return nil
}
