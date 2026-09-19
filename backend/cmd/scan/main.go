// Command scan checks a single URL and prints the finding as JSON. It is the smallest
// way to see what the scanner makes of a page, without a database or a queue.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/praetorianer777/behoerdenbarriere/internal/config"
	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
)

func main() {
	url := flag.String("url", "", "URL to check")
	links := flag.Bool("links", false, "also print the links found")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "usage: scan -url https://www.example.de")
		os.Exit(2)
	}
	if err := run(*url, *links); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(url string, withLinks bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s, err := scanner.New(ctx, scanner.Options{
		ChromeURL:   os.Getenv("CHROME_URL"),
		PageTimeout: cfg.Scan.PageTimeout,
	})
	if err != nil {
		return err
	}
	defer s.Close()

	got := s.Scan(ctx, url)
	got.Result.IsEntry = true

	out := struct {
		Result any      `json:"result"`
		Grade  string   `json:"grade,omitempty"`
		Links  []string `json:"links,omitempty"`
	}{Result: got.Result}

	// A page that did not load has no score. Grading it would turn a failed request
	// into a perfect result — 100 points for a page nobody ever saw.
	if !got.Result.Failed() {
		got.Result.Score = scoring.PageScore(got.Result)
		out.Result = got.Result
		out.Grade = scoring.Grade(got.Result.Score)
	}
	if withLinks {
		out.Links = got.Links
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	if got.Result.Failed() {
		return fmt.Errorf("page could not be checked: %s", got.Result.Err)
	}
	return nil
}
