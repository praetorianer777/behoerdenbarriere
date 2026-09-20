package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/usage"
)

type fakeRecorder struct{ hits []usage.Hit }

func (f *fakeRecorder) Record(hit usage.Hit) { f.hits = append(f.hits, hit) }

func visit(t *testing.T, srv *Server, method, path string, header map[string]string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = "203.0.113.7:51234"
	req.Header.Set("User-Agent", "Mozilla/5.0 Firefox/130")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	srv.Routes().ServeHTTP(httptest.NewRecorder(), req)
}

func TestEndpointsAreCountedByPatternNotByPath(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewServer(&fakeDB{agencies: []store.AgencyListing{}}, Options{}).WithUsage(rec)

	visit(t, srv, http.MethodGet, "/api/v1/agencies/stadt-kiel", nil)

	if len(rec.hits) != 1 {
		t.Fatalf("%d hits, want 1", len(rec.hits))
	}
	if want := "GET /api/v1/agencies/{slug}"; rec.hits[0].Endpoint != want {
		t.Errorf("endpoint = %q, want %q", rec.hits[0].Endpoint, want)
	}
}

func TestPageViewComesFromTheHeader(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewServer(&fakeDB{}, Options{}).WithUsage(rec)

	visit(t, srv, http.MethodPost, "/api/v1/view", map[string]string{
		"X-Page": "/behoerde/stadt-kiel",
	})

	if len(rec.hits) != 1 {
		t.Fatalf("%d hits, want 1", len(rec.hits))
	}
	hit := rec.hits[0]
	if hit.Page != "/behoerde/:slug" || hit.Slug != "stadt-kiel" {
		t.Errorf("page = %q, slug = %q", hit.Page, hit.Slug)
	}
}

// A path nobody can reach must not become a row; otherwise the table can be filled
// from outside.
func TestUnknownPageIsNotCounted(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewServer(&fakeDB{}, Options{}).WithUsage(rec)

	visit(t, srv, http.MethodPost, "/api/v1/view", map[string]string{"X-Page": "/erfunden"})

	if len(rec.hits) != 1 {
		t.Fatalf("%d hits, want 1", len(rec.hits))
	}
	if rec.hits[0].Page != "" {
		t.Errorf("page = %q, want nothing counted", rec.hits[0].Page)
	}
}

// Reading the statistics must not change them.
func TestCounterDoesNotCountItself(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewServer(&fakeDB{}, Options{}).WithUsage(rec)

	visit(t, srv, http.MethodGet, "/api/v1/usage", nil)
	visit(t, srv, http.MethodPost, "/api/v1/view", nil)

	for _, hit := range rec.hits {
		if hit.Endpoint != "" {
			t.Errorf("own endpoint counted as %q", hit.Endpoint)
		}
	}
}

func TestHealthChecksAreNotCounted(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewServer(&fakeDB{}, Options{}).WithUsage(rec)

	visit(t, srv, http.MethodGet, "/healthz", nil)
	visit(t, srv, http.MethodGet, "/readyz", nil)

	if len(rec.hits) != 0 {
		t.Fatalf("%d hits for the health checks, want none", len(rec.hits))
	}
}

func TestAPIWorksWithoutACounter(t *testing.T) {
	rec := do(t, NewServer(&fakeDB{}, Options{}), http.MethodGet, "/api/v1/agencies")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestViewAnswersEmpty(t *testing.T) {
	srv := NewServer(&fakeDB{}, Options{}).WithUsage(&fakeRecorder{})
	res := httptest.NewRecorder()
	srv.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/view", nil))
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", res.Code)
	}
	if res.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", res.Body.String())
	}
}

func TestUsageEndpoint(t *testing.T) {
	day := time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)
	db := &fakeDB{usage: &store.UsageSummary{
		Since: day.AddDate(0, 0, -6), Until: day, Visitors: 12, Views: 40,
		Days:     []store.UsageDay{{Day: day, Visitors: 5, Views: 17}},
		Pages:    []store.UsageKey{{Key: "/", Count: 30}},
		Agencies: []store.UsageKey{{Key: "stadt-kiel", Count: 8}},
	}}
	srv := NewServer(db, Options{})

	rec := do(t, srv, http.MethodGet, "/api/v1/usage?days=7")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if db.usageDays != 7 {
		t.Errorf("window = %d days, want 7", db.usageDays)
	}

	var body struct {
		Since, Until string
		Visitors     int64
		Views        int64
		Days         []struct {
			Day      string
			Visitors int64
		}
		Endpoints []struct{ Key string }
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if body.Until != "2026-09-19" || body.Since != "2026-09-13" {
		t.Errorf("window = %s..%s", body.Since, body.Until)
	}
	if body.Visitors != 12 || body.Views != 40 {
		t.Errorf("visitors = %d, views = %d", body.Visitors, body.Views)
	}
	if len(body.Days) != 1 || body.Days[0].Day != "2026-09-19" {
		t.Errorf("days = %+v", body.Days)
	}
	if body.Endpoints == nil {
		t.Error("endpoints missing from the answer; an empty list is expected")
	}
}

func TestUsageWindowIsBounded(t *testing.T) {
	db := &fakeDB{}
	srv := NewServer(db, Options{})

	do(t, srv, http.MethodGet, "/api/v1/usage?days=99999")
	if db.usageDays != 365 {
		t.Errorf("window = %d days, want it capped at 365", db.usageDays)
	}
	do(t, srv, http.MethodGet, "/api/v1/usage?days=-3")
	if db.usageDays != 1 {
		t.Errorf("window = %d days, want at least 1", db.usageDays)
	}
}
