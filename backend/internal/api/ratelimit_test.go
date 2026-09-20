package api

import (
	"testing"
	"time"
)

func TestBucketRefillsOverTime(t *testing.T) {
	now := time.Now()
	l := newLimiter(time.Hour)
	l.now = func() time.Time { return now }
	r := perMinute(60, 2)

	if !l.allow("a", r).ok || !l.allow("a", r).ok {
		t.Fatal("burst of 2 was not granted")
	}
	d := l.allow("a", r)
	if d.ok {
		t.Fatal("third request inside a second was granted")
	}
	if d.retryAfter <= 0 || d.retryAfter > time.Second {
		t.Errorf("retryAfter = %v, want at most a second", d.retryAfter)
	}

	now = now.Add(time.Second)
	if !l.allow("a", r).ok {
		t.Fatal("a token per second was not refilled")
	}
}

func TestBucketsAreSeparatePerKey(t *testing.T) {
	l := newLimiter(time.Hour)
	r := perMinute(60, 1)
	if !l.allow("a", r).ok || !l.allow("b", r).ok {
		t.Fatal("one key used up another key's budget")
	}
}

// Keys come from the network, so the map must not grow with every client that shows
// up once. An idle bucket is a full bucket, dropping it changes nothing.
func TestIdleBucketsAreEvicted(t *testing.T) {
	now := time.Now()
	l := newLimiter(time.Minute)
	l.now = func() time.Time { return now }
	r := perMinute(60, 1)

	for i := range 10 {
		l.allow(string(rune('a'+i)), r)
	}
	if l.size() != 10 {
		t.Fatalf("buckets = %d, want 10", l.size())
	}

	now = now.Add(2 * time.Minute)
	l.allow("fresh", r)
	if l.size() != 1 {
		t.Fatalf("buckets after idle time = %d, want only the fresh one", l.size())
	}
}

func TestZeroRateMeansNoLimit(t *testing.T) {
	l := newLimiter(time.Hour)
	for range 100 {
		if d := l.allow("a", rate{}); !d.ok || d.limit != 0 {
			t.Fatalf("unlimited rate rejected: %+v", d)
		}
	}
}

func TestResetCountsUpToAFullBucket(t *testing.T) {
	l := newLimiter(time.Hour)
	d := l.allow("a", perMinute(60, 10))
	if seconds(d.reset) != 1 {
		t.Errorf("reset = %v, want the second it takes to refill one token", d.reset)
	}
}
