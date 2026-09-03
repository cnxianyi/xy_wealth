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
	if cfg.Binance.FuturesBaseURL != "https://fapi.binance.com" {
		t.Fatalf("Futures base URL = %q, want https://fapi.binance.com", cfg.Binance.FuturesBaseURL)
	}
	if cfg.Binance.CoinMFuturesBaseURL != "https://dapi.binance.com" {
		t.Fatalf("COIN-M Futures base URL = %q, want https://dapi.binance.com", cfg.Binance.CoinMFuturesBaseURL)
	}
	if cfg.Weex.SpotBaseURL != "https://api-spot.weex.com" {
		t.Fatalf("Weex Spot base URL = %q, want https://api-spot.weex.com", cfg.Weex.SpotBaseURL)
	}
	if cfg.Weex.ContractBaseURL != "https://api-contract.weex.com" {
		t.Fatalf("Weex Contract base URL = %q, want https://api-contract.weex.com", cfg.Weex.ContractBaseURL)
	}
}
