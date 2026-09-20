// Command seed loads the list of authorities into the database. It is idempotent:
// running it again updates master data and adds new entries, and never removes one.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/seed"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	agencies, err := seed.Load()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		return err
	}
	for _, a := range agencies {
		if _, err := db.UpsertAgency(ctx, a); err != nil {
			return err
		}
	}

	total, err := db.CountAgencies(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("%d authorities loaded, %d on record\n", len(agencies), total)
	return nil
}
