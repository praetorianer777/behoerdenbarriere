package api

import (
	"math"
	"sync"
	"time"
)

// rate describes a token bucket: how fast it refills and how many requests may
// arrive at once.
type rate struct {
	perSecond float64
	burst     int
}

func perMinute(n, burst int) rate {
	return rate{perSecond: float64(n) / 60, burst: burst}
}

func (r rate) valid() bool { return r.perSecond > 0 && r.burst > 0 }

type bucket struct {
	tokens float64
	last   time.Time
}

// decision carries what the caller needs for the X-RateLimit-* headers.
type decision struct {
	ok         bool
	limit      int
	remaining  int
	reset      time.Duration
	retryAfter time.Duration
}

// limiter keeps one token bucket per key. The keys come from the network, so the map
// would grow with every new client; buckets idle for longer than idleTTL are dropped,
// which is safe because an idle bucket is a full bucket and a fresh one starts full.
type limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	idleTTL   time.Duration
	lastSweep time.Time
	now       func() time.Time
}

const sweepAfterBuckets = 1024

func newLimiter(idleTTL time.Duration) *limiter {
	if idleTTL <= 0 {
		idleTTL = 10 * time.Minute
	}
	return &limiter{buckets: map[string]*bucket{}, idleTTL: idleTTL, now: time.Now}
}

func (l *limiter) allow(key string, r rate) decision {
	if !r.valid() {
		return decision{ok: true}
	}
	burst := float64(r.burst)
	full := time.Duration(burst / r.perSecond * float64(time.Second))

	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(burst, b.tokens+now.Sub(b.last).Seconds()*r.perSecond)
	b.last = now

	d := decision{limit: r.burst}
	if b.tokens >= 1 {
		b.tokens--
		d.ok = true
	} else {
		d.retryAfter = time.Duration((1 - b.tokens) / r.perSecond * float64(time.Second))
	}
	d.remaining = int(b.tokens)
	d.reset = min(time.Duration((burst-b.tokens)/r.perSecond*float64(time.Second)), full)
	return d
}

func (l *limiter) sweep(now time.Time) {
	if len(l.buckets) < sweepAfterBuckets && now.Sub(l.lastSweep) < l.idleTTL {
		return
	}
	l.lastSweep = now
	for key, b := range l.buckets {
		if now.Sub(b.last) > l.idleTTL {
			delete(l.buckets, key)
		}
	}
}

func (l *limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
