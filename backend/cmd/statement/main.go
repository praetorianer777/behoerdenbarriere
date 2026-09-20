// Command statement checks the accessibility statement of a single site.
//
// It exists for two reasons: to look at one authority without running a whole scan,
// and to hold ourselves to the same check in the pipeline. A project that asks 426
// authorities for their statement should be able to show its own.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
	"github.com/praetorianer777/behoerdenbarriere/internal/statement"
)

func main() {
	url := flag.String("url", "", "address of the statement")
	strict := flag.Bool("strict", false, "exit non-zero unless every required element is there")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "usage: statement -url https://www.example.de/erklaerung-zur-barrierefreiheit")
		os.Exit(2)
	}
	if err := run(*url, *strict); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(url string, strict bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s, err := scanner.New(ctx, scanner.Options{
		ChromeURL:   os.Getenv("CHROME_URL"),
		PageTimeout: 60 * time.Second,
	})
	if err != nil {
		return err
	}
	defer s.Close()

	scan := s.Scan(ctx, url)
	if scan.Result.Failed() {
		return fmt.Errorf("the page could not be read: %s", scan.Result.Err)
	}

	// Depth 1 stands for "reached from the start page": the page was asked for
	// directly here, and the link on the start page is what the caller is checking.
	result := statement.Check(statement.Input{
		Pages: []statement.Page{{URL: url, Text: scan.Result.Text, Depth: 1}},
	})

	fmt.Printf("%s\n%d von %d Pflichtangaben gefunden\n\n", url, result.Met(), len(result.Findings))
	for _, finding := range result.Findings {
		mark := "fehlt   "
		if finding.Met {
			mark = "vorhanden"
		}
		fmt.Printf("  %-10s %s\n", mark, finding.Requirement)
		if finding.Evidence != "" {
			fmt.Printf("             %s\n", finding.Evidence)
		}
	}

	if strict && !result.Complete() {
		return fmt.Errorf("%d von %d Pflichtangaben fehlen", len(result.Findings)-result.Met(), len(result.Findings))
	}
	return nil
}
