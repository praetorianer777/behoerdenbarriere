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
