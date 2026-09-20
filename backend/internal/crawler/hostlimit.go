package crawler

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// HostLimiter keeps the promise of one request per second per host — across every scan
// running at the same time, not per scan.
//
// The distinction matters because it is not an edge case: every federal ministry lives
// under bund.de. bmi.bund.de and bmf.bund.de are two authorities to us and one machine
// to them, and a limiter created per crawl would quietly double the load on that
// machine the moment both are scanned together.
//
// The limiter is keyed by host rather than by registrable domain. Two hosts of a state
// portal are usually two servers, and slowing all of Berlin down to one request per
// second because berlin.de is one domain would be politeness nobody asked for.
type HostLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func NewHostLimiter() *HostLimiter {
	return &HostLimiter{limiters: map[string]*rate.Limiter{}}
}

// Wait blocks until the host may be asked again. A slower interval than the one already
// in force wins: robots.txt asking for five seconds must not be undone by another scan
// of the same host that read nothing.
func (h *HostLimiter) Wait(ctx context.Context, host string, interval time.Duration) error {
	return h.limiter(host, interval).Wait(ctx)
}

func (h *HostLimiter) limiter(host string, interval time.Duration) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()

	limiter, known := h.limiters[host]
	if !known {
		limiter = rate.NewLimiter(rate.Every(interval), 1)
		h.limiters[host] = limiter
		return limiter
	}
	if wanted := rate.Every(interval); wanted < limiter.Limit() {
		limiter.SetLimit(wanted)
	}
	return limiter
}
