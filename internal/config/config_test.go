package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsAndEnvironmentOverrides(t *testing.T) {
	t.Setenv("XY_WEALTH_HTTP_ADDRESS", ":9090")
	t.Setenv("XY_WEALTH_LOG_LEVEL", "debug")
	t.Setenv("XY_WEALTH_POSTGRES_MAX_OPEN_CONNS", "42")
	t.Setenv("XY_WEALTH_BINANCE_INCLUDE_ZERO", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Address != ":9090" {
		t.Fatalf("HTTP address = %q, want :9090", cfg.HTTP.Address)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("log level = %q, want debug", cfg.Log.Level)
	}
	if cfg.Postgres.MaxOpenConns != 42 {
		t.Fatalf("max open connections = %d, want 42", cfg.Postgres.MaxOpenConns)
	}
	if !cfg.Binance.IncludeZero {
		t.Fatal("Binance IncludeZero = false, want true")
	}
	if cfg.HTTP.ShutdownTimeout != 10*time.Second {
		t.Fatalf("shutdown timeout = %s, want 10s", cfg.HTTP.ShutdownTimeout)
	}
}
