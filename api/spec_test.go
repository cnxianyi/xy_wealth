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
		"/api/v1/summary",
		"/api/v1/summary/exchanges",
		"/api/v1/summary/banks",
	} {
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("OpenAPI document is missing path %q", path)
		}
	}
}
