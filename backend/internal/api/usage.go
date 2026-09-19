package api

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/praetorianer777/behoerdenbarriere/internal/usage"
)

// Recorder is the counting side of the usage statistics, an interface so the
// middleware can be tested without a database behind it.
type Recorder interface {
	Record(hit usage.Hit)
}

// pageHeader carries the path of the website the request was made from. The browser
// cannot be asked afterwards which page it is showing, and guessing it from the
// endpoint would be wrong as soon as two pages use the same data.
const pageHeader = "X-Page"

// silentPaths are the counter's own endpoints. Counting them would mean that reading
// the statistics changes them.
var silentPaths = map[string]bool{
	"/api/v1/usage": true,
	"/api/v1/view":  true,
}

// usageCounter counts after the request was served, because only then chi knows which
// route matched — the endpoint is counted by its pattern, so an authority's slug never
// becomes part of an endpoint key.
func (s *Server) usageCounter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if s.usage == nil {
			return
		}

		hit := usage.Hit{
			At:        time.Now(),
			IP:        clientIP(r),
			UserAgent: r.UserAgent(),
		}
		if !silentPaths[r.URL.Path] {
			if pattern := chi.RouteContext(r.Context()).RoutePattern(); pattern != "" {
				hit.Endpoint = r.Method + " " + pattern
			}
		}
		if page, slug, ok := usage.NormalizePage(r.Header.Get(pageHeader)); ok {
			hit.Page, hit.Slug = page, slug
		}
		s.usage.Record(hit)
	})
}

// handleView is the page view of a page that needs no data of its own. It answers
// nothing; the counting happened in the middleware.
func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	days := min(max(atoi(r.URL.Query().Get("days"), 30), 1), 365)
	summary, err := s.db.Usage(r.Context(), days)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUsageDTO(*summary))
}

// clientIP is only ever used to build the daily hash and is never stored. RealIP has
// already applied the proxy headers by the time this runs.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
