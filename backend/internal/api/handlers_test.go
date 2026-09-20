package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/model"
	"github.com/praetorianer777/behoerdenbarriere/internal/scoring"
	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/trend"
)

func ptr(v float64) *float64 { return &v }

func request(t *testing.T, db Queries, method, path string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	testServer(db).Routes().ServeHTTP(rec, req)
	return rec
}

// testServer is the API as it runs in production, with the default limits and one
// configured key.
func testServer(db Queries) *Server {
	return NewServer(db, Options{APIKeys: []string{"geheim"}, Limits: DefaultLimits()})
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, rec.Body.String())
	}
	return out
}

func sampleAgencies() []store.AgencyListing {
	scanned := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	return []store.AgencyListing{
		{
			Agency: model.Agency{ID: 1, Slug: "bmi", Name: "Bundesministerium des Innern",
				URL: "https://www.bmi.bund.de/", Level: model.LevelBund},
			Score: ptr(74.5), Grade: "C", ScannedAt: &scanned, Pages: 42, PrevScore: ptr(70.5),
			Perceivable: ptr(68), Operable: ptr(80), Understandable: ptr(90), Robust: ptr(85),
		},
		{
			Agency: model.Agency{ID: 2, Slug: "stadt-kiel", Name: "Kiel",
				URL: "https://www.kiel.de/", Level: model.LevelKommune, State: "Schleswig-Holstein"},
			Score: nil, Grade: "",
		},
	}
}

func TestAgencyListCarriesScoreAndDelta(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies(), total: 2}
	rec := request(t, db, http.MethodGet, "/api/v1/agencies", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode[listDTO](t, rec)
	if got.Total != 2 || len(got.Items) != 2 {
		t.Fatalf("list = %+v", got)
	}
	if got.Items[0].Delta == nil || *got.Items[0].Delta != 4 {
		t.Errorf("delta = %v, want 4", got.Items[0].Delta)
	}
	// An authority that has not been scanned has no score — and no delta either. A
	// zero would read as "checked, scored nothing".
	if got.Items[1].Score != nil || got.Items[1].Delta != nil {
		t.Errorf("unscanned authority: %+v", got.Items[1])
	}
}

func TestAgencyListPassesFiltersThrough(t *testing.T) {
	db := &fakeDB{}
	request(t, db, http.MethodGet,
		"/api/v1/agencies?q=kiel&level=kommune&state=Schleswig-Holstein&grade=C&sort=score_asc&page=3&per_page=10", nil)

	want := store.AgencyFilter{Query: "kiel", Level: "kommune", State: "Schleswig-Holstein",
		Grade: "C", Sort: "score_asc", Page: 3, PerPage: 10}
	if db.filter != want {
		t.Fatalf("filter = %+v, want %+v", db.filter, want)
	}
}

// Nonsense in the query string must not turn into an error page; the ranking falls
// back to its defaults.
func TestAgencyListIgnoresUnusablePaging(t *testing.T) {
	db := &fakeDB{}
	rec := request(t, db, http.MethodGet, "/api/v1/agencies?page=abc&per_page=-5", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if db.filter.Page != 1 || db.filter.PerPage != -5 {
		t.Fatalf("filter = %+v", db.filter)
	}
}

func TestAgencyListEmptyIsAnEmptyArray(t *testing.T) {
	rec := request(t, &fakeDB{}, http.MethodGet, "/api/v1/agencies", nil)
	if rec.Body.String() == "" {
		t.Fatal("empty body")
	}
	// A client should be able to iterate without checking for null first.
	got := decode[map[string]any](t, rec)
	if got["items"] == nil {
		t.Fatalf("items = null: %s", rec.Body.String())
	}
}

func TestAgencyDetailCarriesTrendAndSubscores(t *testing.T) {
	now := time.Now()
	db := &fakeDB{
		agencies: sampleAgencies(),
		scanIDs:  []int64{99},
		history: []trend.Point{
			{At: now.AddDate(0, 0, -60), Score: 60, Grade: "D"},
			{At: now.AddDate(0, 0, -30), Score: 67, Grade: "D"},
			{At: now.AddDate(0, 0, -1), Score: 74.5, Grade: "C"},
		},
	}
	rec := request(t, db, http.MethodGet, "/api/v1/agencies/bmi", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := decode[agencyDetailDTO](t, rec)
	if got.Slug != "bmi" || got.LatestID != 99 {
		t.Errorf("detail = %+v", got.agencyDTO)
	}
	if got.Subscores.Perceivable == nil || *got.Subscores.Perceivable != 68 {
		t.Errorf("subscores = %+v", got.Subscores)
	}
	if len(got.History) != 3 {
		t.Errorf("%d history points", len(got.History))
	}
	if got.Trend.Direction != trend.Improved {
		t.Errorf("direction = %s", got.Trend.Direction)
	}
	if got.Trend.PointsPerMonth == nil {
		t.Error("no pace although there are three scans")
	}
}

// An authority that has never been scanned has to answer, not fail — it is exactly
// the case the ranking wants to show.
func TestAgencyDetailWithoutScans(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	rec := request(t, db, http.MethodGet, "/api/v1/agencies/stadt-kiel", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode[agencyDetailDTO](t, rec)
	if got.LatestID != 0 || got.Trend.Direction != trend.Unknown {
		t.Fatalf("detail = %+v", got.Trend)
	}
}

func TestUnknownAgencyIsNotFound(t *testing.T) {
	rec := request(t, &fakeDB{agencies: sampleAgencies()}, http.MethodGet, "/api/v1/agencies/gibtsnicht", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func sampleScan() *store.ScanDetail {
	finished := time.Date(2026, 9, 18, 9, 30, 0, 0, time.UTC)
	return &store.ScanDetail{
		ID: 99, AgencyID: 1, AgencySlug: "bmi", AgencyName: "BMI", Status: "done",
		StartedAt: finished.Add(-10 * time.Minute), FinishedAt: &finished,
		Score: ptr(74.5), Grade: "C", PagesScanned: 42, PagesFailed: 1,
		Perceivable: ptr(68), Operable: ptr(80), Understandable: ptr(90), Robust: ptr(85),
	}
}

func rule(id string, impact model.Impact) scoring.RuleSummary {
	return scoring.RuleSummary{RuleID: id, Impact: impact, Principle: model.Perceivable, Pages: 2, Nodes: 5}
}

func TestLatestScanShowsWhatChanged(t *testing.T) {
	db := &fakeDB{
		agencies: sampleAgencies(),
		scanIDs:  []int64{99, 98},
		scan:     sampleScan(),
		rules: map[int64][]scoring.RuleSummary{
			98: {rule("image-alt", model.ImpactCritical), rule("color-contrast", model.ImpactSerious)},
			99: {rule("color-contrast", model.ImpactSerious), rule("label", model.ImpactCritical)},
		},
		pages: []store.PageDetail{{URL: "https://www.bmi.bund.de/", IsEntry: true, Score: ptr(70)}},
	}

	rec := request(t, db, http.MethodGet, "/api/v1/agencies/bmi/scans/latest", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := decode[scanDTO](t, rec)
	if got.ID != 99 || got.Grade != "C" || got.PagesFailed != 1 {
		t.Errorf("scan = %+v", got)
	}
	if len(got.Rules) != 2 || len(got.Pages) != 1 {
		t.Errorf("%d rules, %d pages", len(got.Rules), len(got.Pages))
	}
	if got.Changes == nil {
		t.Fatal("no comparison with the previous scan")
	}
	if len(got.Changes.Fixed) != 1 || got.Changes.Fixed[0].RuleID != "image-alt" {
		t.Errorf("fixed = %+v", got.Changes.Fixed)
	}
	if len(got.Changes.Introduced) != 1 || got.Changes.Introduced[0].RuleID != "label" {
		t.Errorf("introduced = %+v", got.Changes.Introduced)
	}
}

// The first scan has nothing to compare against, and the answer must not pretend
// otherwise.
func TestLatestScanWithoutAPreviousOne(t *testing.T) {
	db := &fakeDB{
		agencies: sampleAgencies(),
		scanIDs:  []int64{99},
		scan:     sampleScan(),
		rules:    map[int64][]scoring.RuleSummary{99: {rule("label", model.ImpactCritical)}},
	}
	rec := request(t, db, http.MethodGet, "/api/v1/agencies/bmi/scans/latest", nil)
	got := decode[scanDTO](t, rec)
	if got.Changes != nil {
		t.Fatalf("comparison out of nothing: %+v", got.Changes)
	}
}

func TestLatestScanWithoutAnyScan(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	rec := request(t, db, http.MethodGet, "/api/v1/agencies/bmi/scans/latest", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

// The number has to be able to say what it is made of; that is the whole point of
// publishing it.
func TestLatestScanExplainsTheScore(t *testing.T) {
	db := &fakeDB{
		agencies: sampleAgencies(),
		scanIDs:  []int64{99},
		scan:     sampleScan(),
		rules:    map[int64][]scoring.RuleSummary{99: {rule("image-alt", model.ImpactCritical)}},
		pageResults: []model.PageResult{
			{URL: "https://www.bmi.bund.de/", IsEntry: true, DOMNodes: 800, Violations: []model.Violation{
				{RuleID: "image-alt", Impact: model.ImpactCritical, Principle: model.Perceivable, NodeCount: 12},
				{RuleID: "region", Impact: model.ImpactModerate, Principle: model.Robust, NodeCount: 1},
			}},
			{URL: "https://www.bmi.bund.de/kontakt", Priority: true, DOMNodes: 600},
		},
	}

	rec := request(t, db, http.MethodGet, "/api/v1/agencies/bmi/scans/latest", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := decode[scanDTO](t, rec)
	if got.Explanation == nil {
		t.Fatal("no explanation")
	}
	if len(got.Explanation.Pages) != 2 {
		t.Fatalf("%d pages explained", len(got.Explanation.Pages))
	}

	entry := got.Explanation.Pages[0]
	if entry.Weight != 3 {
		t.Errorf("entry page weight = %v", entry.Weight)
	}
	if len(entry.Reasons) != 2 || entry.Reasons[0].RuleID != "image-alt" {
		t.Errorf("reasons = %+v", entry.Reasons)
	}
	if entry.Reasons[0].PointsIfFixed <= 0 {
		t.Errorf("fixing the worst finding gains nothing: %+v", entry.Reasons[0])
	}
	if len(got.Explanation.Improvements) == 0 ||
		got.Explanation.Improvements[0].RuleID != "image-alt" {
		t.Errorf("improvements = %+v", got.Explanation.Improvements)
	}
}

func TestScanByID(t *testing.T) {
	db := &fakeDB{scan: sampleScan(), rules: map[int64][]scoring.RuleSummary{}}
	rec := request(t, db, http.MethodGet, "/api/v1/scans/99", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode[scanDTO](t, rec); got.ID != 99 {
		t.Fatalf("scan = %+v", got)
	}

	if rec := request(t, db, http.MethodGet, "/api/v1/scans/abc", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("status for a non-numeric id = %d", rec.Code)
	}
	if rec := request(t, db, http.MethodGet, "/api/v1/scans/12345", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("status for an unknown scan = %d", rec.Code)
	}
}

func TestStats(t *testing.T) {
	db := &fakeDB{
		stats: &store.Stats{
			Agencies: 92, Scanned: 80, AvgScore: ptr(64.2),
			Grades:  map[string]int{"C": 30, "F": 20},
			ByLevel: []store.GroupScore{{Name: "bund", Agencies: 36, AvgScore: ptr(70)}},
			ByState: []store.GroupScore{{Name: "Bayern", Agencies: 4, AvgScore: ptr(61)}},
			TopRules: []store.RuleCount{
				{RuleID: "color-contrast", Impact: "serious", Agencies: 70, Pages: 900},
			},
		},
		states: []string{"Bayern", "Hessen"},
	}

	rec := request(t, db, http.MethodGet, "/api/v1/stats", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode[statsDTO](t, rec)
	if got.Agencies != 92 || got.Scanned != 80 || got.AvgScore == nil {
		t.Errorf("stats = %+v", got)
	}
	if got.Grades["C"] != 30 || len(got.ByLevel) != 1 || len(got.ByState) != 1 {
		t.Errorf("breakdown = %+v", got)
	}
	if len(got.TopRules) != 1 || got.TopRules[0].RuleID != "color-contrast" {
		t.Errorf("rules = %+v", got.TopRules)
	}
	if len(got.States) != 2 {
		t.Errorf("states = %v", got.States)
	}
}

// A rescan costs the authority traffic, so it is not open to everyone.
func TestRescanNeedsTheAPIKey(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}

	if rec := request(t, db, http.MethodPost, "/api/v1/agencies/bmi/rescan", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("without a key: status = %d", rec.Code)
	}
	if rec := request(t, db, http.MethodPost, "/api/v1/agencies/bmi/rescan",
		map[string]string{"X-API-Key": "falsch"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("with a wrong key: status = %d", rec.Code)
	}
	if len(db.queued) != 0 {
		t.Fatalf("queued anyway: %v", db.queued)
	}

	rec := request(t, db, http.MethodPost, "/api/v1/agencies/bmi/rescan",
		map[string]string{"X-API-Key": "geheim"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("with the right key: status = %d", rec.Code)
	}
	if len(db.queued) != 1 || db.queued[0] != 1 {
		t.Fatalf("queue = %v", db.queued)
	}
}

// An unconfigured key must not let everyone in.
func TestRescanRefusedWhenNoKeyIsConfigured(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agencies/bmi/rescan", nil)
	NewServer(db, Options{Limits: DefaultLimits()}).Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

// A database error must not reach the caller as a message; it could carry internals.
func TestDatabaseErrorStaysInside(t *testing.T) {
	db := &fakeDB{failWith: errBoom}
	rec := request(t, db, http.MethodGet, "/api/v1/agencies", nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "database unreachable") {
		t.Fatalf("internals leaked: %s", body)
	}
}
