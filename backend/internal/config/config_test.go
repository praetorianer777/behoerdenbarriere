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

func TestLoadRejectsInvalidValues(t *testing.T) {
	cases := map[string]string{
		"CRAWL_MAX_PAGES":    "many",
		"CRAWL_RATE_PER_SEC": "fast",
		"CRAWL_TIMEOUT":      "soon",
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
