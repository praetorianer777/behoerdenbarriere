package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func do(t *testing.T, srv *Server, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	rec := do(t, NewServer(fakePinger{}, "", ""), http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v", body)
	}
}

// healthz must not depend on the database: otherwise a database restart would restart
// the container too and drag the outage out.
func TestHealthzIndependentOfDatabase(t *testing.T) {
	rec := do(t, NewServer(fakePinger{err: errors.New("down")}, "", ""), http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestReadyzReflectsDatabase(t *testing.T) {
	ok := do(t, NewServer(fakePinger{}, "", ""), http.MethodGet, "/readyz")
	if ok.Code != http.StatusOK {
		t.Errorf("healthy database: status = %d", ok.Code)
	}
	down := do(t, NewServer(fakePinger{err: errors.New("no connection")}, "", ""), http.MethodGet, "/readyz")
	if down.Code != http.StatusServiceUnavailable {
		t.Errorf("broken database: status = %d", down.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	srv := NewServer(fakePinger{}, "http://localhost:5173", "")
	rec := do(t, srv, http.MethodGet, "/healthz")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("origin header = %q", got)
	}

	preflight := do(t, srv, http.MethodOptions, "/healthz")
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", preflight.Code)
	}
}

func TestNoCORSHeaderWithoutConfiguredOrigin(t *testing.T) {
	rec := do(t, NewServer(fakePinger{}, "", ""), http.MethodGet, "/healthz")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected origin header: %q", got)
	}
}
