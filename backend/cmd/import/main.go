// Command import adds authorities from Wikidata to the ones kept by hand.
//
// The hand-kept list is checked and stays authoritative: where both know the same
// authority, the checked entry stands. The import only fills the breadth — the roughly
// three hundred districts nobody wants to maintain by hand.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/wikidata"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "only report what would be imported")
	timeout := flag.Duration("timeout", 15*time.Minute, "budget for the whole import")
	flag.Parse()

	if err := run(*dryRun, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(dryRun bool, timeout time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	agencies, fetchErr := wikidata.New("", 90*time.Second).FetchDistricts(ctx)
	if len(agencies) == 0 {
		return fmt.Errorf("nothing came back: %w", fetchErr)
	}
	fmt.Printf("%d districts from Wikidata\n", len(agencies))

	if dryRun {
		for _, a := range agencies {
			fmt.Printf("  %-34s %-24s %s\n", a.Slug, a.State, a.URL)
		}
		return reportGaps(fetchErr)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		return err
	}

	var added, known int
	for _, a := range agencies {
		_, isNew, err := db.UpsertImported(ctx, a)
		if err != nil {
			return err
		}
		if isNew {
			added++
		} else {
			known++
		}
	}

	total, err := db.CountAgencies(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("%d added, %d already known, %d authorities on record\n", added, known, total)
	return reportGaps(fetchErr)
}

// reportGaps keeps a partial import from passing as a complete one.
func reportGaps(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("the import is incomplete — %w", err)
}
