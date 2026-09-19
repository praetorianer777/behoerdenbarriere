package usage

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSink struct {
	counts  []Count
	dropped []time.Time
	fail    error
}

func (f *fakeSink) AddUsage(_ context.Context, counts []Count) error {
	if f.fail != nil {
		return f.fail
	}
	f.counts = append(f.counts, counts...)
	return nil
}

func (f *fakeSink) DropVisitorHashes(_ context.Context, before time.Time) error {
	f.dropped = append(f.dropped, before)
	return nil
}

func (f *fakeSink) total(kind Kind, key string) int64 {
	var sum int64
	for _, c := range f.counts {
		if c.Kind == kind && c.Key == key {
			sum += c.Count
		}
	}
	return sum
}

func (f *fakeSink) keys(kind Kind) map[string]int64 {
	out := map[string]int64{}
	for _, c := range f.counts {
		if c.Kind == kind {
			out[c.Key] += c.Count
		}
	}
	return out
}

var now = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

func TestRecordAggregatesInsteadOfLoggingEvents(t *testing.T) {
	sink := &fakeSink{}
	rec := NewRecorder(sink, 30)

	for range 5 {
		rec.Record(Hit{At: now, IP: "203.0.113.7", UserAgent: "Mozilla/5.0",
			Endpoint: "GET /api/v1/agencies/{slug}", Page: "/behoerde/:slug", Slug: "kiel"})
	}
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	if got := sink.total(KindPage, "/behoerde/:slug"); got != 5 {
		t.Errorf("page views = %d, want 5", got)
	}
	if got := sink.total(KindAgency, "kiel"); got != 5 {
		t.Errorf("authority views = %d, want 5", got)
	}
	if got := sink.total(KindAPI, "GET /api/v1/agencies/{slug}"); got != 5 {
		t.Errorf("endpoint calls = %d, want 5", got)
	}
	if visitors := sink.keys(KindVisitor); len(visitors) != 1 {
		t.Errorf("%d visitor rows for one visitor, want 1", len(visitors))
	}
}

func TestVisitorsAreToldApartWithinADay(t *testing.T) {
	sink := &fakeSink{}
	rec := NewRecorder(sink, 30)

	rec.Record(Hit{At: now, IP: "203.0.113.7", UserAgent: "Mozilla/5.0 Firefox"})
	rec.Record(Hit{At: now, IP: "203.0.113.8", UserAgent: "Mozilla/5.0 Firefox"})
	rec.Record(Hit{At: now, IP: "203.0.113.7", UserAgent: "Mozilla/5.0 Safari"})
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	if visitors := sink.keys(KindVisitor); len(visitors) != 3 {
		t.Fatalf("%d visitors, want 3", len(visitors))
	}
}

// The whole point of the daily salt: the same person is a different hash tomorrow, so
// the rows cannot be joined into a history.
func TestVisitorHashDoesNotSurviveTheDay(t *testing.T) {
	sink := &fakeSink{}
	rec := NewRecorder(sink, 30)

	hit := Hit{At: now, IP: "203.0.113.7", UserAgent: "Mozilla/5.0"}
	rec.Record(hit)
	hit.At = now.AddDate(0, 0, 1)
	rec.Record(hit)
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	visitors := sink.keys(KindVisitor)
	if len(visitors) != 2 {
		t.Fatalf("%d hashes across two days, want 2 different ones", len(visitors))
	}
}

func TestNothingIdentifyingReachesTheSink(t *testing.T) {
	sink := &fakeSink{}
	rec := NewRecorder(sink, 30)
	rec.Record(Hit{At: now, IP: "203.0.113.7", UserAgent: "Mozilla/5.0 Firefox/130"})
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	for _, c := range sink.counts {
		if c.Key == "203.0.113.7" || c.Key == "Mozilla/5.0 Firefox/130" {
			t.Fatalf("address or user agent written as key: %q", c.Key)
		}
	}
}

func TestBotsAreNotCounted(t *testing.T) {
	sink := &fakeSink{}
	rec := NewRecorder(sink, 30)
	for _, ua := range []string{"", "Googlebot/2.1", "curl/8.5.0", "UptimeRobot/2.0"} {
		rec.Record(Hit{At: now, IP: "203.0.113.7", UserAgent: ua, Page: "/"})
	}
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(sink.counts) != 0 {
		t.Fatalf("%d counters written for bots, want none", len(sink.counts))
	}
}

func TestFailedFlushKeepsTheCounts(t *testing.T) {
	sink := &fakeSink{fail: errors.New("database gone")}
	rec := NewRecorder(sink, 30)
	rec.Record(Hit{At: now, IP: "203.0.113.7", UserAgent: "Mozilla/5.0", Page: "/"})

	if err := rec.Flush(context.Background()); err == nil {
		t.Fatal("flush reported success although the sink failed")
	}
	sink.fail = nil
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatalf("second flush: %v", err)
	}
	if got := sink.total(KindPage, "/"); got != 1 {
		t.Fatalf("page views after retry = %d, want 1", got)
	}
}

func TestOldVisitorHashesArePruned(t *testing.T) {
	sink := &fakeSink{}
	rec := NewRecorder(sink, 30)
	rec.Record(Hit{At: now, IP: "203.0.113.7", UserAgent: "Mozilla/5.0"})
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	if len(sink.dropped) != 1 {
		t.Fatalf("%d prunes, want 1", len(sink.dropped))
	}
	if want := DayOf(now).AddDate(0, 0, -30); !sink.dropped[0].Equal(want) {
		t.Errorf("pruned before %v, want %v", sink.dropped[0], want)
	}
}

func TestNormalizePage(t *testing.T) {
	cases := []struct {
		in, page, slug string
		ok             bool
	}{
		{in: "/", page: "/", ok: true},
		{in: "/dashboard", page: "/dashboard", ok: true},
		{in: "/statistik/", page: "/statistik", ok: true},
		{in: "/behoerde/stadt-kiel", page: "/behoerde/:slug", slug: "stadt-kiel", ok: true},
		{in: ""},
		{in: "/erfunden"},
		{in: "https://anderswo.example/"},
		{in: "/behoerde/Groß Ünsinn"},
		{in: "/behoerde/a/b"},
	}
	for _, c := range cases {
		page, slug, ok := NormalizePage(c.in)
		if ok != c.ok || page != c.page || slug != c.slug {
			t.Errorf("NormalizePage(%q) = %q, %q, %v; want %q, %q, %v",
				c.in, page, slug, ok, c.page, c.slug, c.ok)
		}
	}
}
