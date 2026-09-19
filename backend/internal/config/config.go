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
	APIAddr     string
	APIKey      string
	CORSOrigin  string
	LogLevel    string

	API    APIConfig
	Crawl  CrawlConfig
	Scan   ScanConfig
	Worker WorkerConfig
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

type CrawlConfig struct {
	MaxPages   int
	MaxDepth   int
	RatePerSec float64
	Timeout    time.Duration
}

type ScanConfig struct {
	PageTimeout time.Duration
	Concurrency int
}

type WorkerConfig struct {
	PollInterval   time.Duration
	RescanInterval time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		DatabaseURL: env("DATABASE_URL", "postgres://behoerdenbarriere:behoerdenbarriere@localhost:5432/behoerdenbarriere?sslmode=disable"),
		ChromeURL:   env("CHROME_URL", "http://localhost:9222"),
		APIAddr:     env("API_ADDR", ":8080"),
		APIKey:      env("API_KEY", ""),
		CORSOrigin:  env("CORS_ORIGIN", "http://localhost:5173"),
		LogLevel:    env("LOG_LEVEL", "info"),
	}

	var err error
	if c.API, err = loadAPI(); err != nil {
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
	if c.Scan.Concurrency, err = envInt("SCAN_CONCURRENCY", 4); err != nil {
		return nil, err
	}
	if c.Worker.PollInterval, err = envDuration("WORKER_POLL_INTERVAL", 5*time.Second); err != nil {
		return nil, err
	}
	if c.Worker.RescanInterval, err = envDuration("RESCAN_INTERVAL", 168*time.Hour); err != nil {
		return nil, err
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
