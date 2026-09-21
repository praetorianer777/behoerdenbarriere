// Package api serves the HTTP interface.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/praetorianer777/behoerdenbarriere/internal/store"
	"github.com/praetorianer777/behoerdenbarriere/internal/version"
)

// Limits are the guard rails of the public API. The data is public and meant to be
// used in bulk; the limits only keep one client from taking the site down for
// everyone, so they are generous and every one of them can be turned off with a zero.
type Limits struct {
	ReadPerMinute      int
	ReadBurst          int
	ExpensivePerMinute int
	ExpensiveBurst     int
	KeyPerMinute       int
	KeyBurst           int
	TrustedProxies     []netip.Prefix
	RescanPerAgency    time.Duration
	MaxBodyBytes       int64
	HistoryPoints      int
	ListItems          int
	CacheMaxAge        time.Duration
	BucketIdleTTL      time.Duration
	RequestTimeout     time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		ReadPerMinute:      120,
		ReadBurst:          60,
		ExpensivePerMinute: 20,
		ExpensiveBurst:     10,
		KeyPerMinute:       600,
		KeyBurst:           200,
		RescanPerAgency:    time.Hour,
		MaxBodyBytes:       64 << 10,
		HistoryPoints:      200,
		ListItems:          500,
		CacheMaxAge:        5 * time.Minute,
		BucketIdleTTL:      10 * time.Minute,
		RequestTimeout:     30 * time.Second,
	}
}

type Options struct {
	CORSOrigin string
	APIKeys    []string
	Limits     Limits
	Operator   Operator
}

type Server struct {
	db            Queries
	corsOrigin    string
	apiKeys       []string
	limits        Limits
	operator      Operator
	limiter       *limiter
	rescanLimiter *limiter
	usage         Recorder
	log           *slog.Logger
}

// NewServer builds the API. A zero rate in Options.Limits switches that particular
// rate limit off, which is what the tests that are not about limits rely on; the size
// caps have no "off", an unbounded response is never wanted.
func NewServer(db Queries, opts Options) *Server {
	limits := opts.Limits
	if limits.HistoryPoints <= 0 {
		limits.HistoryPoints = DefaultLimits().HistoryPoints
	}
	if limits.ListItems <= 0 {
		limits.ListItems = DefaultLimits().ListItems
	}
	return &Server{
		db:            db,
		corsOrigin:    opts.CORSOrigin,
		apiKeys:       opts.APIKeys,
		limits:        limits,
		operator:      opts.Operator,
		limiter:       newLimiter(limits.BucketIdleTTL),
		rescanLimiter: newLimiter(maxDuration(limits.RescanPerAgency, limits.BucketIdleTTL)),
		log:           slog.Default(),
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

// WithUsage turns the usage counter on. Without it the API serves the same, only
// without counting — the statistics are a feature of the site, not a condition of it.
func (s *Server) WithUsage(recorder Recorder) *Server {
	s.usage = recorder
	return s
}

func (s *Server) Routes() http.Handler {
	read := perMinute(s.limits.ReadPerMinute, s.limits.ReadBurst)
	// The statistics and the rule catalogue aggregate over every scan, so they are the
	// cheapest way to make the database work hard.
	expensive := perMinute(s.limits.ExpensivePerMinute, s.limits.ExpensiveBurst)
	keyed := perMinute(s.limits.KeyPerMinute, s.limits.KeyBurst)

	r := chi.NewRouter()
	// Deliberately without chi's RealIP: it rewrites RemoteAddr from headers no matter
	// who sent them. The limiter needs an identity that cannot be forged, see clientIP.
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(middleware.Timeout(s.limits.RequestTimeout))
	r.Use(limitBody(s.limits.MaxBodyBytes))
	r.Use(s.cors)

	// Health checks stay unlimited: they are what tells us the limits are not the
	// reason the site looks down.
	r.Get("/healthz", s.handleHealth)
	r.Get("/readyz", s.handleReady)

	r.Route("/api/v1", func(r chi.Router) {
		// Counting wraps everything below it, so a request that a limit refuses is
		// not counted as a visit — the numbers should say what was served.
		r.Use(s.usageCounter)

		r.Group(func(r chi.Router) {
			r.Use(s.rateLimit("read", read, keyed), cache(s.limits.CacheMaxAge))
			r.Get("/agencies", s.handleAgencies)
			r.Get("/agencies/{slug}", s.handleAgency)
			r.Get("/agencies/{slug}/scans/latest", s.handleLatestScan)
			r.Get("/scans/{id}", s.handleScan)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.rateLimit("expensive", expensive, keyed), cache(s.limits.CacheMaxAge))
			r.Get("/stats", s.handleStats)
			r.Get("/rules", s.handleRules)
			r.Get("/version", s.handleVersion)
			r.Get("/operator", s.handleOperator)
			r.Get("/thirdparties", s.handleThirdParties)
			r.Get("/mail", s.handleMail)
			r.Get("/usage", s.handleUsage)
		})
		r.Group(func(r chi.Router) {
			// The page ping travels with every navigation, so it belongs on the read
			// budget; caching it would be pointless, it answers nothing.
			r.Use(s.rateLimit("read", read, keyed))
			r.Post("/view", s.handleView)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.rateLimit("write", expensive, keyed))
			r.Post("/agencies/{slug}/rescan", s.handleRescan)
		})
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
// handleHealth also says which build is answering. A tag like "latest" moves, so an
// installation that kept an older copy is indistinguishable from a current one until
// something asks.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"version":  version.Short(),
		"revision": version.Current(),
	})
}

// handleVersion answers the same as /healthz, but on the path the proxy passes
// through: /healthz is reachable inside the network, and the question "which build is
// actually running out there" is asked from outside.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":  version.Short(),
		"revision": version.Current(),
	})
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
