package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string
	ChromeURL   string
	APIAddr     string
	APIKey      string
	CORSOrigin  string
	LogLevel    string

	Crawl  CrawlConfig
	Scan   ScanConfig
	Worker WorkerConfig
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
