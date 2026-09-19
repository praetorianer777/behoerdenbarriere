// Package api serves the HTTP interface.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/praetorianer777/behoerdenbarriere/internal/store"
)

type Server struct {
	db         Queries
	corsOrigin string
	apiKey     string
	usage      Recorder
	log        *slog.Logger
}

func NewServer(db Queries, corsOrigin, apiKey string) *Server {
	return &Server{db: db, corsOrigin: corsOrigin, apiKey: apiKey, log: slog.Default()}
}

// WithUsage turns the usage counter on. Without it the API serves the same, only
// without counting — the statistics are a feature of the site, not a condition of it.
func (s *Server) WithUsage(recorder Recorder) *Server {
	s.usage = recorder
	return s
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(s.cors)

	r.Get("/healthz", s.handleHealth)
	r.Get("/readyz", s.handleReady)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.usageCounter)
		r.Get("/agencies", s.handleAgencies)
		r.Get("/agencies/{slug}", s.handleAgency)
		r.Get("/agencies/{slug}/scans/latest", s.handleLatestScan)
		r.Post("/agencies/{slug}/rescan", s.handleRescan)
		r.Get("/scans/{id}", s.handleScan)
		r.Get("/stats", s.handleStats)
		r.Get("/rules", s.handleRules)
		r.Get("/usage", s.handleUsage)
		r.Post("/view", s.handleView)
	})
	return r
}

// fail turns an error into an answer. Only "not found" is told to the caller; anything
// else could carry internals, so it is logged and answered with a bare 500.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	s.log.Error("request failed", "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// healthz only reports that the process is up; readyz asks the database. They are
// separate so that a database restart does not take the container down with it.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.corsOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.corsOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Page")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
