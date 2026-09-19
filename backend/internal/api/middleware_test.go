package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/trend"
)

func testLimits(l Limits) Limits {
	if l.RequestTimeout == 0 {
		l.RequestTimeout = 5 * time.Second
	}
	if l.BucketIdleTTL == 0 {
		l.BucketIdleTTL = time.Minute
	}
	return l
}

type call struct {
	method string
	path   string
	from   string
	header map[string]string
}

func send(t *testing.T, h http.Handler, c call) *httptest.ResponseRecorder {
	t.Helper()
	method := c.method
	if method == "" {
		method = http.MethodGet
	}
	req := httptest.NewRequest(method, c.path, nil)
	if c.from != "" {
		req.RemoteAddr = c.from
	}
	for k, v := range c.header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReadLimitRejectsWithHeaders(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	h := NewServer(db, Options{Limits: testLimits(Limits{ReadPerMinute: 60, ReadBurst: 2})}).Routes()

	for i := range 2 {
		rec := send(t, h, call{path: "/api/v1/agencies", from: "203.0.113.5:1234"})
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d", i, rec.Code)
		}
		if got := rec.Header().Get("X-RateLimit-Limit"); got != "2" {
			t.Errorf("X-RateLimit-Limit = %q", got)
		}
	}

	rec := send(t, h, call{path: "/api/v1/agencies", from: "203.0.113.5:1234"})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status over the limit = %d", rec.Code)
	}
	for header, want := range map[string]string{
		"X-RateLimit-Limit":     "2",
		"X-RateLimit-Remaining": "0",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if rec.Header().Get("Retry-After") == "" || rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Errorf("Retry-After / X-RateLimit-Reset missing: %v", rec.Header())
	}
}

// One noisy client must not lock everyone else out.
func TestLimitIsPerClientNotGlobal(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	h := NewServer(db, Options{Limits: testLimits(Limits{ReadPerMinute: 60, ReadBurst: 1})}).Routes()

	send(t, h, call{path: "/api/v1/agencies", from: "203.0.113.5:1000"})
	if rec := send(t, h, call{path: "/api/v1/agencies", from: "203.0.113.5:1000"}); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("same client: status = %d", rec.Code)
	}
	if rec := send(t, h, call{path: "/api/v1/agencies", from: "198.51.100.9:1000"}); rec.Code != http.StatusOK {
		t.Fatalf("other client: status = %d", rec.Code)
	}
}

// X-Forwarded-For is a wish, not a fact. Believing it from an untrusted peer would
// hand every client an unlimited supply of identities.
func TestForwardedForOnlyCountsFromATrustedProxy(t *testing.T) {
	spoof := func(t *testing.T, trusted []netip.Prefix, peer string) int {
		db := &fakeDB{agencies: sampleAgencies()}
		h := NewServer(db, Options{Limits: testLimits(Limits{
			ReadPerMinute: 60, ReadBurst: 1, TrustedProxies: trusted,
		})}).Routes()

		send(t, h, call{path: "/api/v1/agencies", from: peer,
			header: map[string]string{"X-Forwarded-For": "198.51.100.1"}})
		rec := send(t, h, call{path: "/api/v1/agencies", from: peer,
			header: map[string]string{"X-Forwarded-For": "198.51.100.2"}})
		return rec.Code
	}

	proxy := []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}
	if code := spoof(t, proxy, "203.0.113.5:1000"); code != http.StatusTooManyRequests {
		t.Errorf("header from an untrusted peer was believed: status = %d", code)
	}
	if code := spoof(t, proxy, "127.0.0.1:1000"); code != http.StatusOK {
		t.Errorf("header from our own proxy was ignored: status = %d", code)
	}
	if code := spoof(t, nil, "127.0.0.1:1000"); code != http.StatusTooManyRequests {
		t.Errorf("without configured proxies the header must not count: status = %d", code)
	}
}

// Behind our own proxy the client is the rightmost address we did not put there
// ourselves; everything further left was written by a hop we do not control.
func TestClientIPTakesTheLastUntrustedHop(t *testing.T) {
	s := NewServer(&fakeDB{}, Options{Limits: Limits{TrustedProxies: []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
	}}})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:5000"
	req.Header.Set("X-Forwarded-For", "9.9.9.9, 198.51.100.7, 10.1.2.3")
	if got := s.clientIP(req); got != "198.51.100.7" {
		t.Errorf("clientIP = %q, want 198.51.100.7", got)
	}
}

func TestStatsHasAStricterLimitThanTheRanking(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	h := NewServer(db, Options{Limits: testLimits(Limits{
		ReadPerMinute: 600, ReadBurst: 50, ExpensivePerMinute: 60, ExpensiveBurst: 1,
	})}).Routes()

	from := "203.0.113.5:1000"
	if rec := send(t, h, call{path: "/api/v1/stats", from: from}); rec.Code != http.StatusOK {
		t.Fatalf("first stats call: status = %d", rec.Code)
	}
	if rec := send(t, h, call{path: "/api/v1/stats", from: from}); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second stats call: status = %d", rec.Code)
	}
	if rec := send(t, h, call{path: "/api/v1/rules", from: from}); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rules shares the strict budget: status = %d", rec.Code)
	}
	if rec := send(t, h, call{path: "/api/v1/agencies", from: from}); rec.Code != http.StatusOK {
		t.Fatalf("the ranking must stay reachable: status = %d", rec.Code)
	}
}

// The whole point of the limit: a refused request costs nothing.
func TestRejectedRequestDoesNoDatabaseWork(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	h := NewServer(db, Options{Limits: testLimits(Limits{
		ReadPerMinute: 60, ReadBurst: 1, ExpensivePerMinute: 60, ExpensiveBurst: 1,
	})}).Routes()

	send(t, h, call{path: "/api/v1/stats", from: "203.0.113.5:1000"})
	before := db.calls
	for range 5 {
		send(t, h, call{path: "/api/v1/stats", from: "203.0.113.5:1000"})
	}
	if db.calls != before {
		t.Fatalf("database was queried %d times for refused requests", db.calls-before)
	}
}

func TestAPIKeyCarriesTheHigherQuota(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	h := NewServer(db, Options{APIKeys: []string{"geheim"}, Limits: testLimits(Limits{
		ReadPerMinute: 60, ReadBurst: 1, KeyPerMinute: 600, KeyBurst: 10,
	})}).Routes()

	keyed := map[string]string{"X-API-Key": "geheim"}
	for i := range 5 {
		if rec := send(t, h, call{path: "/api/v1/agencies", from: "203.0.113.5:1000", header: keyed}); rec.Code != http.StatusOK {
			t.Fatalf("keyed request %d: status = %d", i, rec.Code)
		}
	}
	// A wrong key is not an identity, it falls back to the address.
	wrong := map[string]string{"X-API-Key": "falsch"}
	send(t, h, call{path: "/api/v1/agencies", from: "203.0.113.5:1000", header: wrong})
	if rec := send(t, h, call{path: "/api/v1/agencies", from: "203.0.113.5:1000", header: wrong}); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("wrong key got the higher quota: status = %d", rec.Code)
	}
}

func TestETagAnswersNotModified(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies(), total: 2}
	h := NewServer(db, Options{Limits: testLimits(Limits{CacheMaxAge: 5 * time.Minute})}).Routes()

	first := send(t, h, call{path: "/api/v1/agencies"})
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the answer")
	}
	if got := first.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q", got)
	}

	again := send(t, h, call{path: "/api/v1/agencies", header: map[string]string{"If-None-Match": etag}})
	if again.Code != http.StatusNotModified {
		t.Fatalf("status with a matching ETag = %d", again.Code)
	}
	if again.Body.Len() != 0 {
		t.Errorf("304 carried a body: %q", again.Body.String())
	}
	if again.Header().Get("ETag") != etag {
		t.Errorf("304 without the ETag: %v", again.Header())
	}

	stale := send(t, h, call{path: "/api/v1/agencies", header: map[string]string{"If-None-Match": `"veraltet"`}})
	if stale.Code != http.StatusOK || stale.Body.Len() == 0 {
		t.Fatalf("a stale ETag must get the answer: status = %d", stale.Code)
	}
}

// The answer changes with the data, so the ETag has to as well.
func TestETagFollowsTheContent(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies(), total: 2}
	h := NewServer(db, Options{Limits: testLimits(Limits{CacheMaxAge: time.Minute})}).Routes()

	before := send(t, h, call{path: "/api/v1/agencies"}).Header().Get("ETag")
	db.total = 3
	if after := send(t, h, call{path: "/api/v1/agencies"}).Header().Get("ETag"); after == before {
		t.Fatalf("ETag %q survived a change of the data", after)
	}
}

func TestRescanLimitedPerAgency(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	h := NewServer(db, Options{APIKeys: []string{"geheim"}, Limits: testLimits(Limits{
		KeyPerMinute: 600, KeyBurst: 100, RescanPerAgency: time.Hour,
	})}).Routes()

	keyed := map[string]string{"X-API-Key": "geheim"}
	first := send(t, h, call{method: http.MethodPost, path: "/api/v1/agencies/bmi/rescan", from: "203.0.113.5:1", header: keyed})
	if first.Code != http.StatusAccepted {
		t.Fatalf("first rescan: status = %d", first.Code)
	}
	second := send(t, h, call{method: http.MethodPost, path: "/api/v1/agencies/bmi/rescan", from: "198.51.100.1:1", header: keyed})
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second rescan of the same authority: status = %d", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("no Retry-After on the refused rescan")
	}
	other := send(t, h, call{method: http.MethodPost, path: "/api/v1/agencies/stadt-kiel/rescan", from: "203.0.113.5:1", header: keyed})
	if other.Code != http.StatusAccepted {
		t.Fatalf("another authority: status = %d", other.Code)
	}
	if len(db.queued) != 2 {
		t.Fatalf("queued = %v, want one job per authority", db.queued)
	}
}

func TestHistoryIsCapped(t *testing.T) {
	points := make([]trend.Point, 0, 500)
	for i := range 500 {
		points = append(points, trend.Point{At: time.Now().Add(-time.Duration(i) * time.Hour), Score: 50})
	}
	db := &fakeDB{agencies: sampleAgencies(), history: points}
	h := NewServer(db, Options{Limits: testLimits(Limits{HistoryPoints: 10})}).Routes()

	rec := send(t, h, call{path: "/api/v1/agencies/bmi"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		History []trend.Point `json:"history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if len(got.History) != 10 {
		t.Fatalf("history points = %d, want the cap of 10", len(got.History))
	}
}

func TestOversizedBodyRejected(t *testing.T) {
	db := &fakeDB{agencies: sampleAgencies()}
	h := NewServer(db, Options{APIKeys: []string{"geheim"}, Limits: testLimits(Limits{MaxBodyBytes: 64})}).Routes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agencies/bmi/rescan", strings.NewReader(strings.Repeat("x", 1000)))
	req.Header.Set("X-API-Key", "geheim")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d", rec.Code)
	}
	if db.calls != 0 {
		t.Errorf("database was queried for an oversized request")
	}
}
