package api

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// clientIP decides whom a request is charged to. X-Forwarded-For is only believed
// when the request reached us from a proxy we run ourselves (nginx sits in front of
// the API, see frontend/nginx.conf). Anyone can put that header on a request, so
// trusting it unconditionally would let a single client forge a new identity per
// request and walk past every limit. The trusted set is configured
// (API_TRUSTED_PROXIES) and empty means: believe nobody, use the peer address.
func (s *Server) clientIP(r *http.Request) string {
	peer := peerAddr(r)
	if !s.trusted(peer) {
		return peer.String()
	}
	// Right to left: the rightmost entry that is not one of our own proxies is the
	// closest thing to a real client. Everything further left was written by a hop we
	// do not control.
	forwarded := r.Header.Get("X-Forwarded-For")
	parts := strings.Split(forwarded, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if err != nil {
			continue
		}
		if !s.trusted(addr) {
			return addr.Unmap().String()
		}
	}
	return peer.String()
}

func (s *Server) trusted(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	addr = addr.Unmap()
	for _, p := range s.limits.TrustedProxies {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func peerAddr(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}

// identify returns the bucket key and whether the caller presented a valid API key.
// A key is an identity of its own: it carries the higher quota, and two people behind
// one office NAT should not share a bucket because of it.
func (s *Server) identify(r *http.Request) (string, bool) {
	if key := r.Header.Get("X-API-Key"); key != "" && s.hasKey(key) {
		return "key:" + key, true
	}
	return "ip:" + s.clientIP(r), false
}

func (s *Server) hasKey(key string) bool {
	if key == "" {
		return false
	}
	// Constant time: a plain comparison leaks how much of a key is right through how
	// long it takes to say no, and the answer is worth guessing at — it carries the
	// higher quota and the rescan endpoint.
	found := false
	for _, k := range s.apiKeys {
		if k == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(k), []byte(key)) == 1 {
			found = true
		}
	}
	return found
}

// rateLimit rejects with 429 before the handler runs, so a refused request never
// reaches the database.
func (s *Server) rateLimit(scope string, anonymous, keyed rate) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, hasKey := s.identify(r)
			limit := anonymous
			if hasKey {
				limit = keyed
			}
			d := s.limiter.allow(scope+"|"+id, limit)
			writeLimitHeaders(w, d)
			if !d.ok {
				w.Header().Set("Retry-After", strconv.Itoa(seconds(d.retryAfter)))
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeLimitHeaders(w http.ResponseWriter, d decision) {
	if d.limit == 0 {
		return
	}
	h := w.Header()
	h.Set("X-RateLimit-Limit", strconv.Itoa(d.limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(d.remaining))
	h.Set("X-RateLimit-Reset", strconv.Itoa(seconds(d.reset)))
}

// seconds rounds up, so a client that waits the announced time really has a token.
func seconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int((d + time.Second - 1) / time.Second)
}

func limitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes > 0 {
				// An announced oversized body is refused before it is read, so a
				// sender does not get to push megabytes at us first.
				if r.ContentLength > maxBytes {
					writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
					return
				}
				if r.Body != nil {
					r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

type bufferedWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (b *bufferedWriter) WriteHeader(status int) { b.status = status }

func (b *bufferedWriter) Write(p []byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	return b.body.Write(p)
}

// cache adds Cache-Control and an ETag over the answer. A scan result changes at most
// once a week, so a repeated request should cost neither the database nor the line.
// The ETag is the hash of the body: it stays correct however the answer is built, and
// the answers are small enough to buffer.
func cache(maxAge time.Duration) func(http.Handler) http.Handler {
	control := "public, max-age=" + strconv.Itoa(seconds(maxAge))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			buf := &bufferedWriter{ResponseWriter: w}
			next.ServeHTTP(buf, r)

			status := buf.status
			if status == 0 {
				status = http.StatusOK
			}
			if status != http.StatusOK {
				w.WriteHeader(status)
				_, _ = w.Write(buf.body.Bytes())
				return
			}

			sum := sha256.Sum256(buf.body.Bytes())
			etag := `"` + base64.RawURLEncoding.EncodeToString(sum[:16]) + `"`
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", control)
			if matches(r.Header.Get("If-None-Match"), etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("Content-Length", strconv.Itoa(buf.body.Len()))
			w.WriteHeader(status)
			_, _ = w.Write(buf.body.Bytes())
		})
	}
}

func matches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		if strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
