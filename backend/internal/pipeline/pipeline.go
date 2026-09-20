// Package pipeline ties crawling, checking, scoring and storing together: one run per
// authority, and what comes out of it is a scan in the database.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/praetorianer777/behoerdenbarriere/internal/crawler"
	"github.com/praetorianer777/behoerdenbarriere/internal/lighthouse"
	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/statement"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/telemetry"
)

// Store is the part of the database the pipeline needs.
type Store interface {
	StartScan(ctx context.Context, agencyID int64, cfg map[string]any) (int64, error)
	FinishScan(ctx context.Context, scanID int64, pages []model.PageResult, result scoring.Result) error
	FailScan(ctx context.Context, scanID int64, cause error) error
	SaveLighthouse(ctx context.Context, scanID int64, result store.LighthouseResult) error
	SaveStatement(ctx context.Context, scanID int64, result statement.Result) error
}

// Auditor is the second opinion on the entry page — Google's Lighthouse score, which
// is built the other way round from ours.
type Auditor interface {
	Audit(ctx context.Context, pageURL string) (*lighthouse.Result, error)
}

// Crawler walks a site and returns a result per page.
type Crawler interface {
	Crawl(ctx context.Context, startURL string) (crawler.Outcome, error)
}

type Pipeline struct {
	store   Store
	crawler Crawler
	auditor Auditor
	cfg     crawler.Config
	log     *slog.Logger
}

func New(store Store, c Crawler, cfg crawler.Config, log *slog.Logger) *Pipeline {
	if log == nil {
		log = slog.Default()
	}
	return &Pipeline{store: store, crawler: c, cfg: cfg, log: log}
}

// WithAuditor adds the outside score. Without one the pipeline runs exactly as
// before — the second number is an addition, never a condition.
func (p *Pipeline) WithAuditor(auditor Auditor) *Pipeline {
	p.auditor = auditor
	return p
}

// Result is what one run produced.
type Result struct {
	ScanID int64
	Score  scoring.Result
	Pages  int
}

// Run checks one authority. A crawl that yields no usable page is recorded as a failed
// scan rather than as a score of zero: a site that could not be reached is not a site
// without barriers, and it is not one full of them either.
func (p *Pipeline) Run(ctx context.Context, agency model.Agency) (*Result, error) {
	started := time.Now()

	ctx, span := telemetry.Tracer().Start(ctx, "scan", trace.WithAttributes(
		attribute.Int64("agency.id", agency.ID),
		attribute.String("agency.slug", agency.Slug),
	))
	defer span.End()

	scanID, err := p.store.StartScan(ctx, agency.ID, map[string]any{
		"max_pages": p.cfg.MaxPages,
		"max_depth": p.cfg.MaxDepth,
		"rate":      p.cfg.RatePerSec,
	})
	if err != nil {
		return nil, p.failed(ctx, span, started, 0, err)
	}
	span.SetAttributes(attribute.Int64("scan.id", scanID))

	outcome, err := p.crawl(ctx, agency.URL)
	pages := outcome.Pages
	if err != nil {
		p.fail(ctx, scanID, err)
		return nil, p.failed(ctx, span, started, 0, fmt.Errorf("crawl %s: %w", agency.Slug, err))
	}

	result := p.score(ctx, pages)
	if result.Pages == 0 {
		err := errors.New("no page could be checked")
		p.fail(ctx, scanID, err)
		return nil, p.failed(ctx, span, started, 0, fmt.Errorf("%s: %w", agency.Slug, err))
	}

	if err := p.store.FinishScan(ctx, scanID, pages, result); err != nil {
		return nil, p.failed(ctx, span, started, result.Pages, fmt.Errorf("save %s: %w", agency.Slug, err))
	}

	span.SetAttributes(
		attribute.Int("scan.pages", result.Pages),
		attribute.Float64("scan.score", result.Score),
		attribute.String("scan.grade", result.Grade),
	)
	telemetry.RecordScan(ctx, telemetry.StatusDone, result.Pages, time.Since(started))

	p.checkStatement(ctx, scanID, agency, outcome)
	p.audit(ctx, scanID, agency)

	p.log.InfoContext(ctx, "scan finished",
		"agency", agency.Slug, "score", result.Score, "grade", result.Grade,
		"pages", result.Pages, "duration", time.Since(started).Round(time.Second))

	return &Result{ScanID: scanID, Score: result, Pages: result.Pages}, nil
}

func (p *Pipeline) crawl(ctx context.Context, startURL string) (crawler.Outcome, error) {
	ctx, span := telemetry.Tracer().Start(ctx, "crawl")
	defer span.End()

	outcome, err := p.crawler.Crawl(ctx, startURL)
	span.SetAttributes(
		attribute.Int("crawl.pages", len(outcome.Pages)),
		attribute.Int("crawl.links_seen", len(outcome.SeenLinks)),
	)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return outcome, err
}

func (p *Pipeline) score(ctx context.Context, pages []model.PageResult) scoring.Result {
	_, span := telemetry.Tracer().Start(ctx, "score")
	defer span.End()

	result := scoring.SiteScore(pages)
	span.SetAttributes(attribute.Int("score.pages", result.Pages))
	return result
}

// failed marks the span and counts the scan before the error travels on.
func (p *Pipeline) failed(ctx context.Context, span trace.Span, started time.Time, pages int, err error) error {
	span.SetStatus(codes.Error, err.Error())
	telemetry.RecordScan(ctx, telemetry.StatusFailed, pages, time.Since(started))
	return err
}

// checkStatement reads the accessibility statement out of what the crawl already saw.
// No extra request: the crawler pulls those pages forward anyway, and asking the
// authority's server twice for the same page to answer a legal question would be
// discourteous for no gain.
func (p *Pipeline) checkStatement(ctx context.Context, scanID int64, agency model.Agency, outcome crawler.Outcome) {
	candidates := make([]statement.Page, 0, len(outcome.Pages))
	for _, page := range outcome.Pages {
		if page.Failed() {
			continue
		}
		candidates = append(candidates, statement.Page{
			URL: page.URL, Text: page.Text, Depth: page.Depth,
		})
	}

	result := statement.Check(statement.Input{Pages: candidates, SeenLinks: outcome.SeenLinks})
	if err := p.store.SaveStatement(ctx, scanID, result); err != nil {
		p.log.ErrorContext(ctx, "could not store the statement check", "agency", agency.Slug, "error", err)
		return
	}
	p.log.InfoContext(ctx, "accessibility statement checked",
		"agency", agency.Slug, "state", result.State,
		"met", result.Met(), "of", len(statement.Requirements))
}

// audit asks Lighthouse about the entry page. Only the entry page: a second full
// pass over every crawled page would double the load on the authority for a number
// that is meant as a cross-check, not as a second opinion on every subpage.
//
// A failure here is logged and otherwise ignored. Our own result is already stored,
// and losing it over a missing cross-check would be absurd.
func (p *Pipeline) audit(ctx context.Context, scanID int64, agency model.Agency) {
	if p.auditor == nil {
		return
	}
	result, err := p.auditor.Audit(ctx, agency.URL)
	if err != nil {
		p.log.Warn("lighthouse did not answer", "agency", agency.Slug, "error", err)
		return
	}
	if result == nil {
		return
	}
	if err := p.store.SaveLighthouse(ctx, scanID, store.LighthouseResult{
		Score:  result.Score,
		Failed: result.FailedAudits,
	}); err != nil {
		p.log.Error("could not store the lighthouse score", "agency", agency.Slug, "error", err)
	}
}

func (p *Pipeline) fail(ctx context.Context, scanID int64, cause error) {
	// The scan has to be closed even when the context is already gone; otherwise it
	// stays 'running' and blocks the next one.
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := p.store.FailScan(closeCtx, scanID, cause); err != nil {
		p.log.ErrorContext(closeCtx, "could not mark the scan as failed", "scan", scanID, "error", err)
	}
}
