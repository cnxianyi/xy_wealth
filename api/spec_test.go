package api

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestSpecificationIsOpenAPI31(t *testing.T) {
	var document struct {
		OpenAPI string                    `yaml:"openapi"`
		Paths   map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(Specification(), &document); err != nil {
		t.Fatalf("parse embedded OpenAPI document: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version = %q, want 3.1.0", document.OpenAPI)
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
		"/api/v1/summary",
		"/api/v1/summary/exchanges",
		"/api/v1/summary/banks",
	} {
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("OpenAPI document is missing path %q", path)
		}
	}
}
