package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIAddr != ":8080" {
		t.Errorf("APIAddr = %q", cfg.APIAddr)
	}
	if cfg.Crawl.MaxPages != 100 || cfg.Crawl.MaxDepth != 3 {
		t.Errorf("crawl defaults: %+v", cfg.Crawl)
	}
	if cfg.Crawl.Timeout != 10*time.Minute {
		t.Errorf("Crawl.Timeout = %v", cfg.Crawl.Timeout)
	}
	if cfg.Worker.RescanInterval != 168*time.Hour {
		t.Errorf("RescanInterval = %v", cfg.Worker.RescanInterval)
	}
	if !cfg.Usage.Enabled || cfg.Usage.RetainDays != 30 {
		t.Errorf("usage defaults: %+v", cfg.Usage)
	}
}

func TestUsageCanBeTurnedOff(t *testing.T) {
	t.Setenv("USAGE_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Usage.Enabled {
		t.Fatal("USAGE_ENABLED=false was ignored")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("API_ADDR", ":9999")
	t.Setenv("CRAWL_MAX_PAGES", "7")
	t.Setenv("CRAWL_RATE_PER_SEC", "0.5")
	t.Setenv("SCAN_PAGE_TIMEOUT", "12s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIAddr != ":9999" || cfg.Crawl.MaxPages != 7 ||
		cfg.Crawl.RatePerSec != 0.5 || cfg.Scan.PageTimeout != 12*time.Second {
		t.Fatalf("environment not applied: %+v", cfg)
	}
}

// An empty variable means unset, not zero — otherwise a blank field in a .env file
// would leave the crawler with a page budget of nothing.
func TestEmptyEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("CRAWL_MAX_PAGES", "")
	t.Setenv("API_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Crawl.MaxPages != 100 || cfg.APIAddr != ":8080" {
		t.Fatalf("default not applied: %+v", cfg)
	}
}

func TestAPILimitDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.API.ReadPerMinute != 120 || cfg.API.ReadBurst != 60 {
		t.Errorf("read limit: %+v", cfg.API)
	}
	// The expensive endpoints must be stricter than the ordinary ones, or the
	// distinction buys nothing.
	if cfg.API.ExpensivePerMinute >= cfg.API.ReadPerMinute {
		t.Errorf("expensive limit %d is not stricter than read limit %d",
			cfg.API.ExpensivePerMinute, cfg.API.ReadPerMinute)
	}
	if cfg.API.KeyPerMinute <= cfg.API.ReadPerMinute {
		t.Errorf("a key must be worth more than no key: %+v", cfg.API)
	}
	if cfg.API.RescanPerAgency != time.Hour || cfg.API.MaxBodyBytes != 64*1024 {
		t.Errorf("rescan/body defaults: %+v", cfg.API)
	}
	if cfg.API.HistoryPoints != 200 || cfg.API.ListItems != 500 {
		t.Errorf("size caps: %+v", cfg.API)
	}
	if len(cfg.API.TrustedProxies) == 0 {
		t.Error("no trusted proxies by default, nginx would never be believed")
	}
}

func TestAPILimitsFromEnv(t *testing.T) {
	t.Setenv("API_READ_PER_MINUTE", "10")
	t.Setenv("API_CACHE_MAX_AGE", "90s")
	t.Setenv("API_TRUSTED_PROXIES", "10.1.0.0/16")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.API.ReadPerMinute != 10 || cfg.API.CacheMaxAge != 90*time.Second {
		t.Fatalf("environment not applied: %+v", cfg.API)
	}
	if len(cfg.API.TrustedProxies) != 1 || cfg.API.TrustedProxies[0].String() != "10.1.0.0/16" {
		t.Fatalf("TrustedProxies = %v", cfg.API.TrustedProxies)
	}
}

// Running without a proxy in front must be expressible: then no forwarded header is
// believed at all.
func TestTrustedProxiesCanBeEmptied(t *testing.T) {
	t.Setenv("API_TRUSTED_PROXIES", "none")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.API.TrustedProxies) != 0 {
		t.Fatalf("TrustedProxies = %v", cfg.API.TrustedProxies)
	}
}

func TestAPIKeysCollectBothVariables(t *testing.T) {
	t.Setenv("API_KEY", "operator")
	t.Setenv("API_KEYS", " forschung , presse ,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.APIKeys()
	want := []string{"operator", "forschung", "presse"}
	if len(got) != len(want) {
		t.Fatalf("APIKeys() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("APIKeys() = %v, want %v", got, want)
		}
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	cases := map[string]string{
		"CRAWL_MAX_PAGES":     "many",
		"CRAWL_RATE_PER_SEC":  "fast",
		"CRAWL_TIMEOUT":       "soon",
		"API_READ_PER_MINUTE": "many",
		"API_TRUSTED_PROXIES": "10.1.0.0",
		"USAGE_ENABLED":       "maybe",
	}
	for key, value := range cases {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			if _, err := Load(); err == nil {
				t.Fatalf("%s=%q was accepted", key, value)
			}
		})
	}
}

// Telemetry is off unless an endpoint is set: no collector is needed to run this.
func TestOTelIsOffByDefault(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OTel.Enabled() {
		t.Errorf("telemetry is on without an endpoint: %+v", cfg.OTel)
	}
	if cfg.OTel.Protocol != "grpc" || cfg.OTel.SampleRatio != 1 {
		t.Errorf("OTel defaults: %+v", cfg.OTel)
	}
	if got := cfg.OTel.ServiceNameOr("behoerdenbarriere-api"); got != "behoerdenbarriere-api" {
		t.Errorf("service name = %q", got)
	}
}

func TestOTelFromEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_SERVICE_NAME", "scanner")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.25")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.OTel.Enabled() || cfg.OTel.Protocol != "http/protobuf" || cfg.OTel.SampleRatio != 0.25 {
		t.Fatalf("environment not applied: %+v", cfg.OTel)
	}
	if got := cfg.OTel.ServiceNameOr("behoerdenbarriere-worker"); got != "scanner" {
		t.Fatalf("service name = %q", got)
	}
}

func TestOTelRejectsInvalidValues(t *testing.T) {
	cases := map[string]string{
		"OTEL_EXPORTER_OTLP_PROTOCOL": "carrier pigeon",
		"OTEL_TRACES_SAMPLER_ARG":     "2",
	}
	for key, value := range cases {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			if _, err := Load(); err == nil {
				t.Fatalf("%s=%q was accepted", key, value)
			}
		})
	}
}
