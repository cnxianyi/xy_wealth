package api

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestSpecificationIsOpenAPI31(t *testing.T) {
	var document struct {
		OpenAPI string `yaml:"openapi"`
		Tags    []struct {
			Name        string `yaml:"name"`
			DisplayName string `yaml:"x-displayName"`
			Parent      string `yaml:"parent"`
		} `yaml:"tags"`
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(Specification(), &document); err != nil {
		t.Fatalf("parse embedded OpenAPI document: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version = %q, want 3.1.0", document.OpenAPI)
	}

	tags := make(map[string]struct {
		parent      string
		displayName string
	}, len(document.Tags))
	for _, tag := range document.Tags {
		tags[tag.Name] = struct {
			parent      string
			displayName string
		}{parent: tag.Parent, displayName: tag.DisplayName}
	}
	for name, want := range map[string]struct {
		parent      string
		displayName string
	}{
		"Exchanges":    {displayName: "Exchanges"},
		"Binance":      {parent: "Exchanges", displayName: "Binance"},
		"BinanceSpot":  {parent: "Binance", displayName: "Spot"},
		"BinanceUSDM":  {parent: "Binance", displayName: "USDⓈ-M Futures"},
		"BinanceCoinM": {parent: "Binance", displayName: "COIN-M Futures"},
		"Bitget":       {parent: "Exchanges", displayName: "Bitget"},
		"Weex":         {parent: "Exchanges", displayName: "Weex"},
		"WeexSpot":     {parent: "Weex", displayName: "Spot"},
	} {
		got, ok := tags[name]
		if !ok {
			t.Errorf("OpenAPI document is missing navigation tag %q", name)
			continue
		}
		if got.parent != want.parent || got.displayName != want.displayName {
			t.Errorf("navigation tag %q = parent %q, display %q; want parent %q, display %q", name, got.parent, got.displayName, want.parent, want.displayName)
		}
	}

	for _, path := range []string{
		"/health/live",
		"/health/ready",
		"/api/v1/exchanges/{provider}/balances",
		"/api/v1/exchanges/{provider}/spot/ping",
		"/api/v1/exchanges/{provider}/spot/time",
		"/api/v1/exchanges/{provider}/spot/exchange-info",
		"/api/v1/exchanges/{provider}/spot/depth",
		"/api/v1/exchanges/{provider}/spot/klines",
		"/api/v1/exchanges/{provider}/spot/ticker/24hr",
		"/api/v1/exchanges/{provider}/spot/ticker/price",
		"/api/v1/exchanges/{provider}/spot/ticker/book",
		"/api/v1/exchanges/{provider}/futures/usdm/ping",
		"/api/v1/exchanges/{provider}/futures/usdm/time",
		"/api/v1/exchanges/{provider}/futures/usdm/exchange-info",
		"/api/v1/exchanges/{provider}/futures/usdm/depth",
		"/api/v1/exchanges/{provider}/futures/usdm/klines",
		"/api/v1/exchanges/{provider}/futures/usdm/ticker/24hr",
		"/api/v1/exchanges/{provider}/futures/usdm/ticker/price",
		"/api/v1/exchanges/{provider}/futures/usdm/ticker/book",
		"/api/v1/exchanges/{provider}/futures/usdm/premium-index",
		"/api/v1/exchanges/{provider}/futures/coinm/ping",
		"/api/v1/exchanges/{provider}/futures/coinm/time",
		"/api/v1/exchanges/{provider}/futures/coinm/exchange-info",
		"/api/v1/exchanges/{provider}/futures/coinm/depth",
		"/api/v1/exchanges/{provider}/futures/coinm/klines",
		"/api/v1/exchanges/{provider}/futures/coinm/ticker/24hr",
		"/api/v1/exchanges/{provider}/futures/coinm/ticker/price",
		"/api/v1/exchanges/{provider}/futures/coinm/ticker/book",
		"/api/v1/exchanges/{provider}/futures/coinm/premium-index",
		"/api/v1/summary",
		"/api/v1/summary/exchanges",
		"/api/v1/summary/banks",
	} {
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("OpenAPI document is missing path %q", path)
		}
	}
}
