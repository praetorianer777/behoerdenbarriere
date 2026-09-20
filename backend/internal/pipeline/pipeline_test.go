package pipeline

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/praetorianer777/behoerdenbarriere/internal/crawler"
	"github.com/praetorianer777/behoerdenbarriere/internal/lighthouse"
	"github.com/praetorianer777/behoerdenbarriere/internal/maildns"
	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/statement"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
)

type fakeStore struct {
	scanID    int64
	startErr  error
	finishErr error

	finished      []model.PageResult
	result        scoring.Result
	failed        error
	closed        bool
	statement     *statement.Result
	lighthouse    *store.LighthouseResult
	lighthouseErr error
	mail          *maildns.Record
}

func (f *fakeStore) StartScan(context.Context, int64, map[string]any) (int64, error) {
	return f.scanID, f.startErr
}

func (f *fakeStore) FinishScan(_ context.Context, _ int64, pages []model.PageResult, result scoring.Result) error {
	f.finished = pages
	f.result = result
	f.closed = true
	return f.finishErr
}

func (f *fakeStore) FailScan(_ context.Context, _ int64, cause error) error {
	f.failed = cause
	f.closed = true
	return nil
}

func (f *fakeStore) SaveMail(_ context.Context, _ int64, record maildns.Record) error {
	f.mail = &record
	return nil
}

func (f *fakeStore) SaveStatement(_ context.Context, _ int64, result statement.Result) error {
	f.statement = &result
	return nil
}

func (f *fakeStore) SaveLighthouse(_ context.Context, _ int64, result store.LighthouseResult) error {
	f.lighthouse = &result
	return f.lighthouseErr
}

type fakeCrawler struct {
	pages     []model.PageResult
	seenLinks []string
	err       error
}

func (f *fakeCrawler) Crawl(context.Context, string) (crawler.Outcome, error) {
	return crawler.Outcome{Pages: f.pages, SeenLinks: f.seenLinks}, f.err
}

func agency() model.Agency {
	return model.Agency{ID: 1, Slug: "bmi", Name: "BMI", URL: "https://www.bmi.bund.de/"}
}

func TestRunStoresScoredPages(t *testing.T) {
	store := &fakeStore{scanID: 42}
	pages := []model.PageResult{
		{URL: "https://www.bmi.bund.de/", DOMNodes: 800, IsEntry: true},
		{URL: "https://www.bmi.bund.de/kontakt", DOMNodes: 800, Priority: true, Violations: []model.Violation{
			{RuleID: "label", Impact: model.ImpactCritical, Principle: model.Perceivable, NodeCount: 2},
		}},
	}

	got, err := New(store, &fakeCrawler{pages: pages}, crawler.Config{MaxPages: 10}, nil).
		Run(context.Background(), agency())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.ScanID != 42 || got.Pages != 2 {
		t.Fatalf("result = %+v", got)
	}
	if len(store.finished) != 2 {
		t.Fatalf("%d pages stored", len(store.finished))
	}
	if store.result.Grade == "" || store.result.Score <= 0 {
		t.Fatalf("score not computed: %+v", store.result)
	}
	if store.failed != nil {
		t.Fatalf("scan marked as failed: %v", store.failed)
	}
}

// A site that could not be reached is not a site without barriers — and not one full
// of them either. It has to end up as a failed scan, not as a score.
func TestRunMarksUnreachableSiteAsFailed(t *testing.T) {
	store := &fakeStore{scanID: 7}
	pages := []model.PageResult{{URL: "https://www.bmi.bund.de/", Err: "timeout"}}

	_, err := New(store, &fakeCrawler{pages: pages}, crawler.Config{}, nil).
		Run(context.Background(), agency())
	if err == nil {
		t.Fatal("no error")
	}
	if store.failed == nil {
		t.Fatal("scan was not marked as failed")
	}
	if store.closed != true {
		t.Fatal("scan left open")
	}
}

func TestRunMarksCrawlErrorAsFailed(t *testing.T) {
	store := &fakeStore{scanID: 7}
	boom := errors.New("robots.txt forbids the start page")

	_, err := New(store, &fakeCrawler{err: boom}, crawler.Config{}, nil).
		Run(context.Background(), agency())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
	if !errors.Is(store.failed, boom) {
		t.Fatalf("recorded cause = %v", store.failed)
	}
}

// If a scan of this authority is already running, the pipeline must not open a second
// one — the website would be hit twice at the same time.
func TestRunPassesThroughStartError(t *testing.T) {
	inFlight := errors.New("already running")
	store := &fakeStore{startErr: inFlight}

	_, err := New(store, &fakeCrawler{}, crawler.Config{}, nil).Run(context.Background(), agency())
	if !errors.Is(err, inFlight) {
		t.Fatalf("err = %v, want %v", err, inFlight)
	}
	if store.closed {
		t.Fatal("a scan that was never opened was closed")
	}
}

// A cancelled context must not keep the scan from being closed; otherwise it stays
// 'running' and blocks every later scan of that authority.
func TestRunClosesScanEvenWhenContextIsGone(t *testing.T) {
	store := &fakeStore{scanID: 3}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _ = New(store, &fakeCrawler{err: context.Canceled}, crawler.Config{}, nil).
		Run(ctx, agency())
	if store.failed == nil {
		t.Fatal("scan was not closed")
	}
}

// The spans are what tells a slow authority from a slow scanner, so a run has to
// leave crawl and scoring behind as children of the scan.
func TestRunRecordsSpans(t *testing.T) {
	spans := recordSpans(t)

	pages := []model.PageResult{{URL: "https://www.bmi.bund.de/", DOMNodes: 800, IsEntry: true}}
	_, err := New(&fakeStore{scanID: 7}, &fakeCrawler{pages: pages}, crawler.Config{MaxPages: 10}, nil).
		Run(context.Background(), agency())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	byName := map[string]sdktrace.ReadOnlySpan{}
	for _, span := range spans.Ended() {
		byName[span.Name()] = span
	}
	scan, ok := byName["scan"]
	if !ok {
		t.Fatalf("no scan span: %v", byName)
	}
	for _, child := range []string{"crawl", "score"} {
		span, ok := byName[child]
		if !ok {
			t.Fatalf("no %s span: %v", child, byName)
		}
		if span.Parent().SpanID() != scan.SpanContext().SpanID() {
			t.Errorf("%s is not a child of the scan span", child)
		}
	}
}

func TestFailedRunMarksTheSpan(t *testing.T) {
	spans := recordSpans(t)

	_, err := New(&fakeStore{scanID: 7}, &fakeCrawler{err: errors.New("no route to host")},
		crawler.Config{MaxPages: 10}, nil).Run(context.Background(), agency())
	if err == nil {
		t.Fatal("Run succeeded although the crawl failed")
	}

	for _, span := range spans.Ended() {
		if span.Name() == "scan" && span.Status().Code != codes.Error {
			t.Fatalf("scan span status = %v", span.Status())
		}
	}
}

func recordSpans(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	spans := tracetest.NewSpanRecorder()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	return spans
}

type fakeAuditor struct {
	result *lighthouse.Result
	err    error
	asked  []string
}

func (f *fakeAuditor) Audit(_ context.Context, pageURL string) (*lighthouse.Result, error) {
	f.asked = append(f.asked, pageURL)
	return f.result, f.err
}

func TestRunStoresTheOutsideScore(t *testing.T) {
	st := &fakeStore{scanID: 42}
	auditor := &fakeAuditor{result: &lighthouse.Result{Score: 97, FailedAudits: []string{"color-contrast"}}}
	pages := []model.PageResult{{URL: "https://www.bmi.bund.de/", DOMNodes: 800, IsEntry: true}}

	_, err := New(st, &fakeCrawler{pages: pages}, crawler.Config{}, nil).
		WithAuditor(auditor).
		Run(context.Background(), agency())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st.lighthouse == nil || st.lighthouse.Score != 97 {
		t.Fatalf("outside score = %+v", st.lighthouse)
	}
	// Only the entry page is audited: a second full pass would double the load on the
	// authority for a number that is a cross-check, not a verdict on every subpage.
	if len(auditor.asked) != 1 || auditor.asked[0] != agency().URL {
		t.Fatalf("audited = %v", auditor.asked)
	}
}

// The outside score is an addition. A service that is down, slow or broken must not
// cost us the scan we already have.
func TestRunKeepsTheScanWhenTheAuditFails(t *testing.T) {
	st := &fakeStore{scanID: 42}
	pages := []model.PageResult{{URL: "https://www.bmi.bund.de/", DOMNodes: 800, IsEntry: true}}

	got, err := New(st, &fakeCrawler{pages: pages}, crawler.Config{}, nil).
		WithAuditor(&fakeAuditor{err: errors.New("lighthouse ist abgestürzt")}).
		Run(context.Background(), agency())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !st.closed || got.Score.Grade == "" {
		t.Fatalf("our own result was lost: closed=%v result=%+v", st.closed, got)
	}
	if st.lighthouse != nil {
		t.Fatalf("a score was stored anyway: %+v", st.lighthouse)
	}
}

func TestRunWithoutAnAuditor(t *testing.T) {
	st := &fakeStore{scanID: 42}
	pages := []model.PageResult{{URL: "https://www.bmi.bund.de/", DOMNodes: 800, IsEntry: true}}

	if _, err := New(st, &fakeCrawler{pages: pages}, crawler.Config{}, nil).
		Run(context.Background(), agency()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st.lighthouse != nil {
		t.Fatalf("score without an auditor: %+v", st.lighthouse)
	}
}

// The statement is read out of the pages the crawl already fetched — asking the
// authority's server again for the same page to answer a legal question would be
// discourteous for no gain.
func TestRunChecksTheAccessibilityStatement(t *testing.T) {
	st := &fakeStore{scanID: 42}
	pages := []model.PageResult{
		{URL: "https://www.bmi.bund.de/", DOMNodes: 800, IsEntry: true, Text: "Startseite"},
		{
			URL: "https://www.bmi.bund.de/erklaerung-zur-barrierefreiheit", Depth: 1,
			Priority: true, DOMNodes: 400,
			Text: "Diese Website ist mit der BITV teilweise vereinbar. Nicht barrierefreie Inhalte: " +
				"einige PDF-Dokumente. Erstellt am 14.03.2026. Barrieren melden: barriere@bmi.bund.de. " +
				"Schlichtungsstelle nach § 16 BGG.",
		},
	}

	if _, err := New(st, &fakeCrawler{pages: pages}, crawler.Config{}, nil).
		Run(context.Background(), agency()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st.statement == nil {
		t.Fatal("the statement was not checked")
	}
	if !st.statement.Found() || !st.statement.Complete() {
		t.Fatalf("statement = %+v", st.statement)
	}
}

func TestRunRecordsAMissingStatement(t *testing.T) {
	st := &fakeStore{scanID: 42}
	pages := []model.PageResult{{URL: "https://www.bmi.bund.de/", DOMNodes: 800, IsEntry: true}}

	if _, err := New(st, &fakeCrawler{pages: pages}, crawler.Config{}, nil).
		Run(context.Background(), agency()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st.statement == nil || st.statement.State != statement.StateMissing {
		t.Fatalf("statement = %+v", st.statement)
	}
	// Every requirement is still listed, so the page can name what is missing.
	if len(st.statement.Findings) != len(statement.Requirements) {
		t.Fatalf("%d findings", len(st.statement.Findings))
	}
}

// The RKI links its accessibility statement and its robots.txt forbids the directory
// it sits in. Recording that as "has no statement" would accuse an authority that has
// one; it is our limit, not their failing.
func TestRunSeparatesUnreadableFromMissing(t *testing.T) {
	st := &fakeStore{scanID: 42}
	pages := []model.PageResult{{URL: "https://www.rki.de/", DOMNodes: 800, IsEntry: true}}

	if _, err := New(st, &fakeCrawler{
		pages:     pages,
		seenLinks: []string{"https://www.rki.de/DE/Service/Barrierefreiheit/barrierefreiheit_node.html"},
	}, crawler.Config{}, nil).Run(context.Background(), agency()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if st.statement == nil || st.statement.State != statement.StateUnreadable {
		t.Fatalf("statement = %+v", st.statement)
	}
	if st.statement.URL == "" {
		t.Error("the address we could not read is missing")
	}
}

func goodPages() *fakeCrawler {
	return &fakeCrawler{pages: []model.PageResult{
		{URL: "https://www.musterstadt.de/", DOMNodes: 800, IsEntry: true},
	}}
}

type fakeResolver struct {
	asked  string
	record maildns.Record
}

func (f *fakeResolver) Lookup(_ context.Context, domain string) maildns.Record {
	f.asked = domain
	f.record.Domain = domain
	return f.record
}

// The mail records belong to the domain, not to the web server: mail for a city is
// published under musterstadt.de, not under www.musterstadt.de.
func TestScanRefreshesTheMailRecords(t *testing.T) {
	st := &fakeStore{scanID: 7}
	resolver := &fakeResolver{record: maildns.Record{Provider: maildns.Microsoft365}}
	p := New(st, goodPages(), crawler.Config{}, nil).WithMail(resolver)

	if _, err := p.Run(context.Background(), model.Agency{
		ID: 3, Slug: "musterstadt", URL: "https://www.musterstadt.de/",
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if resolver.asked != "musterstadt.de" {
		t.Errorf("looked up %q, want musterstadt.de", resolver.asked)
	}
	if st.mail == nil || st.mail.Provider != maildns.Microsoft365 {
		t.Errorf("mail record = %+v", st.mail)
	}
}

// Without a resolver nothing changes: the mail records are an addition, never a
// condition for a scan.
func TestScanWithoutAResolverStoresNoMail(t *testing.T) {
	st := &fakeStore{scanID: 7}
	p := New(st, goodPages(), crawler.Config{}, nil)

	if _, err := p.Run(context.Background(), model.Agency{ID: 3, URL: "https://www.musterstadt.de/"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if st.mail != nil {
		t.Errorf("mail record = %+v, want none", st.mail)
	}
}
