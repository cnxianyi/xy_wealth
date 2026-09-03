package binance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/xy-wealth/xy-wealth/internal/config"
)

func TestClientBalances(t *testing.T) {
	const (
		apiKey    = "test-api-key"
		secretKey = "test-secret-key"
		timestamp = int64(1700000000000)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/account" {
			t.Errorf("path = %q, want /api/v3/account", r.URL.Path)
		}
		if got := r.Header.Get("X-MBX-APIKEY"); got != apiKey {
			t.Errorf("API key header = %q, want %q", got, apiKey)
		}
		query := r.URL.Query()
		payload := "recvWindow=" + query.Get("recvWindow") + "&timestamp=" + query.Get("timestamp")
		if got, want := query.Get("signature"), sign(payload, secretKey); got != want {
			t.Errorf("signature = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"balances":[{"asset":"BTC","free":"0.10000000","locked":"0.20000000"},{"asset":"ETH","free":"0","locked":"0"}]}`))
	}))
	defer server.Close()

	client := New(config.BinanceConfig{
		BaseURL:     server.URL,
		APIKey:      apiKey,
		SecretKey:   secretKey,
		RecvWindow:  5000,
		HTTPTimeout: time.Second,
	})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	balances, err := client.Balances(context.Background())
	if err != nil {
		t.Fatalf("Balances() error = %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("balance count = %d, want 1", len(balances))
	}
	if got := balances[0].Total; got != "0.3" {
		t.Fatalf("BTC total = %q, want 0.3", got)
	}
}

func TestClientBalancesRequiresCredentials(t *testing.T) {
	client := New(config.BinanceConfig{BaseURL: "https://example.com", HTTPTimeout: time.Second})
	_, err := client.Balances(context.Background())
	if !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("Balances() error = %v, want ErrCredentialsMissing", err)
	}
}

func TestClientBalancesRedactsSignedURLFromNetworkErrors(t *testing.T) {
	client := New(config.BinanceConfig{
		BaseURL:     "https://example.com",
		APIKey:      "api-key",
		SecretKey:   "secret-key",
		RecvWindow:  5000,
		HTTPTimeout: time.Second,
	})
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, &url.Error{Op: "Get", URL: req.URL.String(), Err: errors.New("network unavailable")}
	})

	_, err := client.Balances(context.Background())
	if err == nil {
		t.Fatal("Balances() error = nil, want network error")
	}
	if strings.Contains(err.Error(), "signature=") {
		t.Fatalf("network error leaked signed URL: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
