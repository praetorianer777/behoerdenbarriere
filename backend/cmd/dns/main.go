// Command dns reads what the authorities' domains publish about their email — MX, SPF
// and DMARC — and stores it.
//
// Only public DNS is read. Nothing is probed, no port is opened, no mail server is
// spoken to.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
)

func main() {
	domain := flag.String("domain", "", "look up a single domain and print it, without a database")
	timeout := flag.Duration("timeout", 30*time.Minute, "budget for the whole run")
	pause := flag.Duration("pause", 100*time.Millisecond, "pause between lookups")
	flag.Parse()

	if err := run(*domain, *timeout, *pause); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(domain string, timeout, pause time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := maildns.New(nil, 15*time.Second)

	if domain != "" {
		print(client.Lookup(ctx, domain))
		return nil
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

	targets, err := db.AgenciesToResolve(ctx)
	if err != nil {
		return err
	}

	var failed int
	for i, target := range targets {
		if i > 0 {
			select {
			case <-time.After(pause):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		host := hostOf(target.URL)
		record := client.Lookup(ctx, host)
		if record.Err != "" {
			failed++
		}
		if err := db.SaveMail(ctx, target.ID, record); err != nil {
			return err
		}
		fmt.Printf("%-40s %-14s %s\n", target.Slug, record.Provider, record.Err)
	}

	fmt.Printf("\n%d authorities, %d lookups without an answer\n", len(targets), failed)
	return nil
}

func print(record maildns.Record) {
	fmt.Printf("domain:   %s\nprovider: %s\n", record.Domain, record.Provider)
	for _, mx := range record.MX {
		fmt.Printf("  MX %3d  %-48s %s\n", mx.Preference, mx.Host, mx.Provider)
	}
	if record.SPF != "" {
		fmt.Printf("SPF:      %s\n", record.SPF)
	}
	if record.DMARC != "" {
		fmt.Printf("DMARC:    %s\n", record.DMARC)
	}
	if record.Err != "" {
		fmt.Printf("error:    %s\n", record.Err)
	}
}

// hostOf takes the domain out of the authority's URL. The mail of a city is published
// under the same domain as its website often enough for this to be the only sensible
// starting point — and where it is not, the record says so itself.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	// "www." is a host of the web server, not of the domain; mail is published one
	// level up.
	return strings.TrimPrefix(u.Hostname(), "www.")
}
