// Package usage counts how the site is used without keeping anything about who used
// it: no cookies, no identifiers handed to the visitor, no raw addresses. What is
// written are aggregate counters per day, and a visitor is only ever a hash under a
// salt that is thrown away with the day.
package usage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Kind string

const (
	KindPage    Kind = "page"
	KindAgency  Kind = "agency"
	KindAPI     Kind = "api"
	KindVisitor Kind = "visitor"
)

// Count is one row of the counter table: how often something happened on one day.
type Count struct {
	Day   time.Time
	Kind  Kind
	Key   string
	Count int64
}

// Hit is one request as the counter sees it. IP and UserAgent are used for the
// visitor hash and then dropped; they are never passed on to the sink.
type Hit struct {
	At        time.Time
	IP        string
	UserAgent string
	Endpoint  string
	Page      string
	Slug      string
}

// Sink is where the counters end up.
type Sink interface {
	AddUsage(ctx context.Context, counts []Count) error
	DropVisitorHashes(ctx context.Context, before time.Time) error
}

type bucket struct {
	day  time.Time
	kind Kind
	key  string
}

// Recorder aggregates in memory and writes batches. A request must not wait for a
// database write, and one row per request would make the table grow with the traffic
// instead of with the days.
type Recorder struct {
	sink   Sink
	log    *slog.Logger
	retain time.Duration

	mu        sync.Mutex
	pending   map[bucket]int64
	salt      []byte
	saltDay   time.Time
	prunedFor time.Time
}

// NewRecorder keeps visitor hashes for retainDays. They cannot be reversed once the
// salt is gone, but a hash nobody needs any more is better deleted than kept.
func NewRecorder(sink Sink, retainDays int) *Recorder {
	if retainDays < 1 {
		retainDays = 1
	}
	return &Recorder{
		sink:    sink,
		log:     slog.Default(),
		retain:  time.Duration(retainDays) * 24 * time.Hour,
		pending: map[bucket]int64{},
	}
}

func (r *Recorder) Record(h Hit) {
	if IsBot(h.UserAgent) {
		return
	}
	day := DayOf(h.At)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.rotateSalt(day)

	if h.Endpoint != "" {
		r.pending[bucket{day, KindAPI, h.Endpoint}]++
	}
	if h.Page != "" {
		r.pending[bucket{day, KindPage, h.Page}]++
	}
	if h.Slug != "" {
		r.pending[bucket{day, KindAgency, h.Slug}]++
	}
	r.pending[bucket{day, KindVisitor, r.visitorHash(h.IP, h.UserAgent)}]++
}

// Run flushes until the context ends, then flushes what is left.
func (r *Recorder) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if err := r.Flush(flushCtx); err != nil {
				r.log.Error("final usage flush failed", "error", err)
			}
			return
		case <-ticker.C:
			if err := r.Flush(ctx); err != nil {
				r.log.Error("usage flush failed", "error", err)
			}
		}
	}
}

// Flush writes the counters collected so far. A failed write puts them back, so a
// database hiccup costs no counts.
func (r *Recorder) Flush(ctx context.Context) error {
	r.mu.Lock()
	pending := r.pending
	r.pending = map[bucket]int64{}
	r.mu.Unlock()

	if len(pending) == 0 {
		return r.prune(ctx, DayOf(time.Now()))
	}

	counts := make([]Count, 0, len(pending))
	var newest time.Time
	for b, n := range pending {
		counts = append(counts, Count{Day: b.day, Kind: b.kind, Key: b.key, Count: n})
		if b.day.After(newest) {
			newest = b.day
		}
	}
	if err := r.sink.AddUsage(ctx, counts); err != nil {
		r.mu.Lock()
		for b, n := range pending {
			r.pending[b] += n
		}
		r.mu.Unlock()
		return err
	}
	return r.prune(ctx, newest)
}

// prune runs once per day, on the first flush that sees a new day.
func (r *Recorder) prune(ctx context.Context, day time.Time) error {
	r.mu.Lock()
	if !day.After(r.prunedFor) {
		r.mu.Unlock()
		return nil
	}
	r.prunedFor = day
	r.mu.Unlock()
	return r.sink.DropVisitorHashes(ctx, day.Add(-r.retain))
}

// rotateSalt draws a new salt whenever the day turns. The old one is overwritten and
// never written down, so yesterday's hashes cannot be recomputed from an address and
// cannot be joined with today's.
func (r *Recorder) rotateSalt(day time.Time) {
	if r.salt != nil && r.saltDay.Equal(day) {
		return
	}
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		panic("usage: no randomness for the daily salt: " + err.Error())
	}
	r.salt, r.saltDay = salt, day
}

func (r *Recorder) visitorHash(ip, userAgent string) string {
	sum := sha256.New()
	sum.Write(r.salt)
	sum.Write([]byte(ip))
	sum.Write([]byte{0})
	sum.Write([]byte(userAgent))
	// Half the digest is far past what is needed to keep visitors apart, and less to
	// keep around.
	return hex.EncodeToString(sum.Sum(nil)[:16])
}

func DayOf(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// botMarkers are the substrings that show up in the user agents of crawlers,
// monitors and command line tools. The list catches the obvious ones; anything that
// hides itself is counted as a visitor, which is the more honest failure.
var botMarkers = []string{
	"bot", "crawler", "spider", "slurp", "crawling", "archiver", "monitoring",
	"headlesschrome", "phantomjs", "curl/", "wget/", "python-requests", "python-urllib",
	"go-http-client", "java/", "okhttp", "libwww-perl", "httpclient", "axios/",
	"facebookexternalhit", "preview", "feedfetcher", "pingdom", "uptimerobot",
	"lighthouse", "pagespeed", "ahrefs", "semrush", "mj12", "dataprovider",
}

// IsBot also treats an empty user agent as a bot: a browser always sends one.
func IsBot(userAgent string) bool {
	if strings.TrimSpace(userAgent) == "" {
		return true
	}
	ua := strings.ToLower(userAgent)
	for _, marker := range botMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,78}[a-z0-9])?$`)

// NormalizePage maps a path of the website onto the key it is counted under. Only
// known routes pass, so nobody can blow the table up with made-up paths, and the
// authority slug is counted on its own instead of inside the page key.
func NormalizePage(path string) (page, slug string, ok bool) {
	if path == "" {
		return "", "", false
	}
	if path = strings.TrimSuffix(path, "/"); path == "" {
		return "/", "", true
	}
	switch path {
	case "/dashboard", "/methodik", "/statistik":
		return path, "", true
	}
	if rest, found := strings.CutPrefix(path, "/behoerde/"); found && slugPattern.MatchString(rest) {
		return "/behoerde/:slug", rest, true
	}
	return "", "", false
}
