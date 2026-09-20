package config

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL string
	ChromeURL   string
	// LighthouseURL points at the service that measures Google's score. Empty means
	// the cross-check is simply not taken.
	LighthouseURL     string
	LighthouseTimeout time.Duration
	APIAddr           string
	APIKey            string
	CORSOrigin        string
	LogLevel          string

	API    APIConfig
	Usage  UsageConfig
	Crawl  CrawlConfig
	Scan   ScanConfig
	Worker WorkerConfig
	OTel   OTelConfig
}

// OTelConfig switches OpenTelemetry on. Without an endpoint nothing is exported and
// no provider is installed, so a local run needs no collector.
type OTelConfig struct {
	Endpoint    string
	Protocol    string
	ServiceName string
	SampleRatio float64
}

func (o OTelConfig) Enabled() bool { return o.Endpoint != "" }

// ServiceNameOr lets each binary name itself while OTEL_SERVICE_NAME still wins.
func (o OTelConfig) ServiceNameOr(def string) string {
	if o.ServiceName != "" {
		return o.ServiceName
	}
	return def
}

// APIConfig holds the guard rails of the public interface. The data is public and
// should stay usable in bulk, so the limits are generous; they exist so that a single
// client cannot take the site down for everyone.
type APIConfig struct {
	ReadPerMinute      int
	ReadBurst          int
	ExpensivePerMinute int
	ExpensiveBurst     int
	KeyPerMinute       int
	KeyBurst           int
	// TrustedProxies are the networks whose X-Forwarded-For header is believed.
	// Empty means: believe nobody. The default covers loopback and the private
	// ranges, because the API only ever sees nginx from the compose network.
	TrustedProxies  []netip.Prefix
	RescanPerAgency time.Duration
	MaxBodyBytes    int64
	HistoryPoints   int
	ListItems       int
	CacheMaxAge     time.Duration
	BucketIdleTTL   time.Duration
	RequestTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
}

// UsageConfig steers the usage counter. FlushInterval is how long counts wait in
// memory before they are written; RetainDays how long the daily visitor hashes are
// kept before they are deleted.
type UsageConfig struct {
	Enabled       bool
	FlushInterval time.Duration
	RetainDays    int
}

type CrawlConfig struct {
	MaxPages   int
	MaxDepth   int
	RatePerSec float64
	Timeout    time.Duration
}

type ScanConfig struct {
	PageTimeout time.Duration
}

type WorkerConfig struct {
	// Concurrency is how many authorities are checked at the same time.
	Concurrency    int
	PollInterval   time.Duration
	RescanInterval time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:   env("DATABASE_URL", "postgres://behoerdenbarriere:behoerdenbarriere@localhost:5432/behoerdenbarriere?sslmode=disable"),
		ChromeURL:     env("CHROME_URL", "http://localhost:9222"),
		LighthouseURL: env("LIGHTHOUSE_URL", ""),
		APIAddr:       env("API_ADDR", ":8080"),
		APIKey:        env("API_KEY", ""),
		CORSOrigin:    env("CORS_ORIGIN", "http://localhost:5173"),
		LogLevel:      env("LOG_LEVEL", "info"),
	}

	var err error
	if c.LighthouseTimeout, err = envDuration("LIGHTHOUSE_TIMEOUT", 3*time.Minute); err != nil {
		return nil, err
	}
	if c.API, err = loadAPI(); err != nil {
		return nil, err
	}
	if c.Usage.Enabled, err = envBool("USAGE_ENABLED", true); err != nil {
		return nil, err
	}
	if c.Usage.FlushInterval, err = envDuration("USAGE_FLUSH_INTERVAL", 10*time.Second); err != nil {
		return nil, err
	}
	if c.Usage.RetainDays, err = envInt("USAGE_RETAIN_DAYS", 30); err != nil {
		return nil, err
	}
	if c.Crawl.MaxPages, err = envInt("CRAWL_MAX_PAGES", 100); err != nil {
		return nil, err
	}
	if c.Crawl.MaxDepth, err = envInt("CRAWL_MAX_DEPTH", 3); err != nil {
		return nil, err
	}
	if c.Crawl.RatePerSec, err = envFloat("CRAWL_RATE_PER_SEC", 1); err != nil {
		return nil, err
	}
	if c.Crawl.Timeout, err = envDuration("CRAWL_TIMEOUT", 10*time.Minute); err != nil {
		return nil, err
	}
	if c.Scan.PageTimeout, err = envDuration("SCAN_PAGE_TIMEOUT", 30*time.Second); err != nil {
		return nil, err
	}
	// Eine Prüfung wartet fast die ganze Zeit — auf den Takt gegenüber der Behörde,
	// auf das Nachladen der Seite. Mehrere gleichzeitig kosten deshalb kaum Rechenzeit,
	// aber jede hält einen Browser-Tab offen; darüber wächst der Bedarf.
	if c.Worker.Concurrency, err = envInt("WORKER_CONCURRENCY", 4); err != nil {
		return c, err
	}
	if c.Worker.PollInterval, err = envDuration("WORKER_POLL_INTERVAL", 5*time.Second); err != nil {
		return nil, err
	}
	if c.Worker.RescanInterval, err = envDuration("RESCAN_INTERVAL", 168*time.Hour); err != nil {
		return nil, err
	}

	c.OTel.Endpoint = env("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	c.OTel.ServiceName = env("OTEL_SERVICE_NAME", "")
	c.OTel.Protocol = env("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	if c.OTel.Protocol != "grpc" && c.OTel.Protocol != "http/protobuf" {
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL: %q is neither grpc nor http/protobuf", c.OTel.Protocol)
	}
	if c.OTel.SampleRatio, err = envFloat("OTEL_TRACES_SAMPLER_ARG", 1); err != nil {
		return nil, err
	}
	if c.OTel.SampleRatio < 0 || c.OTel.SampleRatio > 1 {
		return nil, fmt.Errorf("OTEL_TRACES_SAMPLER_ARG: %v is outside 0..1", c.OTel.SampleRatio)
	}
	return c, nil
}

const defaultTrustedProxies = "127.0.0.0/8,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7"

func loadAPI() (APIConfig, error) {
	a := APIConfig{}
	var err error
	ints := []struct {
		key string
		def int
		dst *int
	}{
		{"API_READ_PER_MINUTE", 120, &a.ReadPerMinute},
		{"API_READ_BURST", 60, &a.ReadBurst},
		{"API_EXPENSIVE_PER_MINUTE", 20, &a.ExpensivePerMinute},
		{"API_EXPENSIVE_BURST", 10, &a.ExpensiveBurst},
		{"API_KEY_PER_MINUTE", 600, &a.KeyPerMinute},
		{"API_KEY_BURST", 200, &a.KeyBurst},
		{"API_HISTORY_POINTS", 200, &a.HistoryPoints},
		{"API_LIST_ITEMS", 500, &a.ListItems},
	}
	for _, f := range ints {
		if *f.dst, err = envInt(f.key, f.def); err != nil {
			return a, err
		}
	}
	durations := []struct {
		key string
		def time.Duration
		dst *time.Duration
	}{
		{"API_RESCAN_PER_AGENCY", time.Hour, &a.RescanPerAgency},
		{"API_CACHE_MAX_AGE", 5 * time.Minute, &a.CacheMaxAge},
		{"API_BUCKET_IDLE_TTL", 10 * time.Minute, &a.BucketIdleTTL},
		{"API_REQUEST_TIMEOUT", 30 * time.Second, &a.RequestTimeout},
		{"API_READ_TIMEOUT", 15 * time.Second, &a.ReadTimeout},
		{"API_WRITE_TIMEOUT", 60 * time.Second, &a.WriteTimeout},
		{"API_IDLE_TIMEOUT", 120 * time.Second, &a.IdleTimeout},
	}
	for _, f := range durations {
		if *f.dst, err = envDuration(f.key, f.def); err != nil {
			return a, err
		}
	}
	maxBody, err := envInt("API_MAX_BODY_BYTES", 64*1024)
	if err != nil {
		return a, err
	}
	a.MaxBodyBytes = int64(maxBody)
	if a.TrustedProxies, err = envPrefixes("API_TRUSTED_PROXIES", defaultTrustedProxies); err != nil {
		return a, err
	}
	return a, nil
}

// APIKeys are the keys that unlock the higher quota. API_KEY stays the key of the
// operator; API_KEYS adds further ones for anyone who asks for bulk access.
func (c *Config) APIKeys() []string {
	var keys []string
	for _, k := range append([]string{c.APIKey}, strings.Split(env("API_KEYS", ""), ",")...) {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func envPrefixes(key, def string) ([]netip.Prefix, error) {
	raw := env(key, def)
	if strings.EqualFold(strings.TrimSpace(raw), "none") {
		return nil, nil
	}
	var out []netip.Prefix
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		p, err := netip.ParsePrefix(part)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func envBool(key string, def bool) (bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func envFloat(key string, def float64) (float64, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}
