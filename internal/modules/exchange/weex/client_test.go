package weex

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

func TestNewUsesV3DomainsAndCredentials(t *testing.T) {
	client := New(config.WeexConfig{
		SpotBaseURL:     "https://spot.example/",
		ContractBaseURL: "https://contract.example/",
		APIKey:          "api-key",
		SecretKey:       "secret-key",
		Passphrase:      "passphrase",
		HTTPTimeout:     3 * time.Second,
	})

	if client.Name() != "weex" {
		t.Fatalf("Name() = %q, want weex", client.Name())
	}
	if client.spotBaseURL != "https://spot.example" || client.contractBaseURL != "https://contract.example" {
		t.Fatalf("base URLs = %q, %q", client.spotBaseURL, client.contractBaseURL)
	}
	if client.apiKey != "api-key" || client.secretKey != "secret-key" || client.passphrase != "passphrase" {
		t.Fatal("credentials were not copied to the client")
	}
	if client.httpClient.Timeout != 3*time.Second {
		t.Fatalf("HTTP timeout = %s, want 3s", client.httpClient.Timeout)
	}
}

func TestNewUsesV3DomainDefaults(t *testing.T) {
	client := New(config.WeexConfig{})
	if client.spotBaseURL != "https://api-spot.weex.com" {
		t.Fatalf("Spot base URL = %q, want https://api-spot.weex.com", client.spotBaseURL)
	}
	if client.contractBaseURL != "https://api-contract.weex.com" {
		t.Fatalf("Contract base URL = %q, want https://api-contract.weex.com", client.contractBaseURL)
	}
	if client.httpClient.Timeout != 10*time.Second {
		t.Fatalf("HTTP timeout = %s, want 10s", client.httpClient.Timeout)
	}
}

func TestClientSatisfiesProviderBoundary(t *testing.T) {
	var provider exchange.Provider = New(config.WeexConfig{})
	_, err := provider.Balances(context.Background())
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Balances() error = %v, want ErrNotImplemented", err)
	}
}
