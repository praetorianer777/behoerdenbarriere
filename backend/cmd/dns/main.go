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
	"sync"
	"syscall"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
)

func main() {
	domain := flag.String("domain", "", "look up a single domain and print it, without a database")
	timeout := flag.Duration("timeout", 2*time.Hour, "budget for the whole run")
	workers := flag.Int("workers", 8, "how many lookups run at the same time")
	flag.Parse()

	if err := run(*domain, *timeout, *workers); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(domain string, timeout time.Duration, workers int) error {
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
	if workers < 1 {
		workers = 1
	}

	// A few lookups at a time. They are almost entirely spent waiting for a resolver,
	// and done one after another a full run takes the better part of an hour — long
	// enough that it stops being run.
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed int
		queue  = make(chan store.MailTarget)
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range queue {
				host := hostOf(target.URL)
				record := client.Lookup(ctx, host)

				mu.Lock()
				if record.Err != "" {
					failed++
				}
				fmt.Printf("%-40s %-14s %s\n", target.Slug, record.Provider, record.Err)
				mu.Unlock()

				if err := db.SaveMail(ctx, target.ID, record); err != nil {
					// A single write that failed is not worth ending the run over; the
					// count at the end says how complete the picture is.
					mu.Lock()
					failed++
					fmt.Fprintln(os.Stderr, err)
					mu.Unlock()
				}
			}
		}()
	}

	for _, target := range targets {
		select {
		case queue <- target:
		case <-ctx.Done():
			close(queue)
			wg.Wait()
			return ctx.Err()
		}
	}
	close(queue)
	wg.Wait()

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
