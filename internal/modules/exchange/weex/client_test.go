package weex

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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
		IncludeZero:     true,
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
	if !client.includeZero {
		t.Fatal("IncludeZero = false, want true")
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

func TestClientSatisfiesSpotProvider(t *testing.T) {
	var provider exchange.SpotProvider = New(config.WeexConfig{})
	if provider.Name() != "weex" {
		t.Fatalf("provider name = %q, want weex", provider.Name())
	}
}

func TestClientSpotPublicEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(accessKeyHeader); got != "" {
			t.Errorf("public request API key header = %q, want empty", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q, want application/json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v3/ping":
			if r.URL.RawQuery != "" {
				t.Errorf("ping query = %q, want empty", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{}`))
		case "/api/v3/time":
			_, _ = w.Write([]byte(`{"serverTime":1700000000123}`))
		case "/api/v3/exchangeInfo":
			if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
				t.Errorf("exchangeInfo symbol = %q, want BTCUSDT", got)
			}
			_, _ = w.Write([]byte(`{"timezone":"UTC","serverTime":1700000000000,"rateLimits":[{"rateLimitType":"REQUEST_WEIGHT","interval":"MINUTE","intervalNum":1,"limit":6000}],"symbols":[{"symbol":"BTCUSDT","status":"online","baseAsset":"BTC","baseAssetPrecision":8,"quoteAsset":"USDT","quoteAssetPrecision":8,"tickSize":0.01,"stepSize":"0.00001","minTradeAmount":"0.0001","maxTradeAmount":100,"takerFeeRate":"0.001","makerFeeRate":0.0005,"buyLimitPriceRatio":"0.1","sellLimitPriceRatio":"0.1","marketBuyLimitSize":"100000","marketSellLimitSize":100000,"marketFallbackPriceRatio":"0.05","enableTrade":true,"enableDisplay":false,"displayDigitMerge":"0.01","displayNew":false,"displayHot":true}]}`))
		case "/api/v3/market/depth":
			query := r.URL.Query()
			if got := query.Get("symbol"); got != "BTCUSDT" {
				t.Errorf("depth symbol = %q, want BTCUSDT", got)
			}
			if got := query.Get("limit"); got != "200" {
				t.Errorf("depth limit = %q, want 200", got)
			}
			_, _ = w.Write([]byte(`{"lastUpdateId":123,"bids":[["100.00","1.50"]],"asks":[["100.10","2.00"]]}`))
		case "/api/v3/market/klines":
			query := r.URL.Query()
			if got := query.Get("symbol"); got != "ETHUSDT" {
				t.Errorf("klines symbol = %q, want ETHUSDT", got)
			}
			if got := query.Get("interval"); got != "1m" {
				t.Errorf("klines interval = %q, want 1m", got)
			}
			if got := query.Get("startTime"); got != "1700000000000" {
				t.Errorf("klines startTime = %q, want 1700000000000", got)
			}
			if got := query.Get("endTime"); got != "1700000060000" {
				t.Errorf("klines endTime = %q, want 1700000060000", got)
			}
			if got := query.Get("timeZone"); got != "" {
				t.Errorf("klines timeZone = %q, want empty because Weex does not support it", got)
			}
			_, _ = w.Write([]byte(`[[1700000000000,"100.00","101.00","99.00","100.50","10.00",1700000059999,"1005.00",42,"5.00","502.50"]]`))
		case "/api/v3/market/ticker/24hr":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","priceChange":"10.00","priceChangePercent":"1.00","lastPrice":"1000.00","bidPrice":"999.90","bidQty":"1.00","askPrice":"1000.10","askQty":"2.00","openPrice":"990.00","highPrice":"1010.00","lowPrice":"980.00","volume":"100.00","quoteVolume":"100500.00","openTime":1700000000000,"closeTime":1700086399999,"count":100}`))
		case "/api/v3/market/ticker/price":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","price":"1000.00"}]`))
		case "/api/v3/market/ticker/bookTicker":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","bidPrice":"999.90","bidQty":"1.00","askPrice":"1000.10","askQty":"2.00"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.WeexConfig{SpotBaseURL: server.URL, HTTPTimeout: time.Second})
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	serverTime, err := client.ServerTime(context.Background())
	if err != nil {
		t.Fatalf("ServerTime() error = %v", err)
	}
	if serverTime.ServerTime != 1700000000123 {
		t.Fatalf("server time = %d, want 1700000000123", serverTime.ServerTime)
	}

	info, err := client.ExchangeInfo(context.Background(), " btcusdt ")
	if err != nil {
		t.Fatalf("ExchangeInfo() error = %v", err)
	}
	if info.Timezone != "UTC" || len(info.Symbols) != 1 || info.Symbols[0].Symbol != "BTCUSDT" {
		t.Fatalf("exchange info = %#v", info)
	}
	if got := info.Symbols[0].TickSize; got != "0.01" {
		t.Fatalf("tick size = %q, want 0.01", got)
	}
	if info.Symbols[0].EnableDisplay == nil || *info.Symbols[0].EnableDisplay {
		t.Fatalf("enable display = %#v, want pointer to false", info.Symbols[0].EnableDisplay)
	}
	if got := info.RateLimits[0].Limit; got != 6000 {
		t.Fatalf("rate limit = %d, want 6000", got)
	}

	orderBook, err := client.Depth(context.Background(), "btcusdt", 200)
	if err != nil {
		t.Fatalf("Depth() error = %v", err)
	}
	if orderBook.LastUpdateID != 123 || !reflect.DeepEqual(orderBook.Bids, [][]string{{"100.00", "1.50"}}) {
		t.Fatalf("order book = %#v", orderBook)
	}

	startTime := int64(1700000000000)
	endTime := int64(1700000060000)
	klines, err := client.Klines(context.Background(), exchange.KlinesRequest{
		Symbol:    "ethusdt",
		Interval:  "1m",
		StartTime: &startTime,
		EndTime:   &endTime,
		TimeZone:  "UTC+8",
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("Klines() error = %v", err)
	}
	if len(klines) != 1 || klines[0].Open != "100.00" || klines[0].NumberOfTrades != 42 || klines[0].Ignore != "" {
		t.Fatalf("klines = %#v", klines)
	}

	ticker, err := client.Ticker24hr(context.Background(), "BTCUSDT")
	if err != nil || ticker.LastPrice != "1000.00" || ticker.WeightedAvgPrice != "1005" {
		t.Fatalf("Ticker24hr() = %#v, error = %v", ticker, err)
	}
	price, err := client.TickerPrice(context.Background(), "BTCUSDT")
	if err != nil || price.Price != "1000.00" {
		t.Fatalf("TickerPrice() = %#v, error = %v", price, err)
	}
	bookTicker, err := client.BookTicker(context.Background(), "BTCUSDT")
	if err != nil || bookTicker.AskPrice != "1000.10" {
		t.Fatalf("BookTicker() = %#v, error = %v", bookTicker, err)
	}
}

func TestClientSpotValidation(t *testing.T) {
	client := New(config.WeexConfig{SpotBaseURL: "https://example.com", HTTPTimeout: time.Second})
	var parameterError exchange.InvalidParameterError

	_, err := client.Depth(context.Background(), "", 15)
	if !errors.As(err, &parameterError) || parameterError.Parameter != "symbol" {
		t.Fatalf("Depth() error = %v, want symbol validation error", err)
	}

	_, err = client.Depth(context.Background(), "BTCUSDT", 20)
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("Depth() error = %v, want limit validation error", err)
	}

	_, err = client.Klines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "3m"})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "interval" {
		t.Fatalf("Klines() error = %v, want interval validation error", err)
	}

	_, err = client.Klines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", Limit: 1001})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("Klines() error = %v, want limit validation error", err)
	}

	start, end := int64(2), int64(1)
	_, err = client.Klines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", StartTime: &start, EndTime: &end})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "startTime" {
		t.Fatalf("Klines() error = %v, want time range validation error", err)
	}
}

func TestClientBalancesAndSignature(t *testing.T) {
	const (
		apiKey     = "test-api-key"
		secretKey  = "test-secret-key"
		passphrase = "test-passphrase"
		timestamp  = int64(1700000000000)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/account" {
			t.Errorf("path = %q, want /api/v3/account", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("signed query = %q, want empty", r.URL.RawQuery)
		}
		if got := r.Header.Get(accessKeyHeader); got != apiKey {
			t.Errorf("API key header = %q, want %q", got, apiKey)
		}
		if got := r.Header.Get(accessTimestampHeader); got != "1700000000000" {
			t.Errorf("timestamp header = %q, want 1700000000000", got)
		}
		if got := r.Header.Get(accessPassphraseHeader); got != passphrase {
			t.Errorf("passphrase header = %q, want %q", got, passphrase)
		}
		wantSignature := sign("1700000000000GET/api/v3/account", secretKey)
		if got := r.Header.Get(accessSignHeader); got != wantSignature {
			t.Errorf("signature = %q, want %q", got, wantSignature)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"balances":[{"asset":"BTC","free":"0.10000000","locked":"0.20000000"},{"asset":"ETH","free":"0","locked":"0"}]}`))
	}))
	defer server.Close()

	client := New(config.WeexConfig{
		SpotBaseURL: server.URL,
		APIKey:      apiKey,
		SecretKey:   secretKey,
		Passphrase:  passphrase,
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

func TestClientBalancesCanIncludeZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"balances":[{"asset":"ETH","free":"0","locked":"0"}]}`))
	}))
	defer server.Close()

	client := New(config.WeexConfig{SpotBaseURL: server.URL, APIKey: "key", SecretKey: "secret", Passphrase: "pass", IncludeZero: true})
	balances, err := client.Balances(context.Background())
	if err != nil {
		t.Fatalf("Balances() error = %v", err)
	}
	if len(balances) != 1 || balances[0].Symbol != "ETH" {
		t.Fatalf("balances = %#v, want zero ETH balance", balances)
	}
}

func TestClientBalancesRequiresCredentials(t *testing.T) {
	client := New(config.WeexConfig{SpotBaseURL: "https://example.com", HTTPTimeout: time.Second})
	_, err := client.Balances(context.Background())
	if !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("Balances() error = %v, want ErrCredentialsMissing", err)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":40012,"message":"Too many requests"}`))
	}))
	defer server.Close()

	client := New(config.WeexConfig{SpotBaseURL: server.URL, HTTPTimeout: time.Second})
	err := client.Ping(context.Background())
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("Ping() error = %v, want APIError", err)
	}
	if apiError.HTTPStatus != http.StatusTooManyRequests || apiError.Code != "40012" || apiError.Message != "Too many requests" {
		t.Fatalf("API error = %#v", apiError)
	}
}

func TestClientNetworkErrorDoesNotExposeSecret(t *testing.T) {
	secret := "super-secret-value"
	client := New(config.WeexConfig{
		SpotBaseURL: "http://127.0.0.1:1",
		APIKey:      "api-key",
		SecretKey:   secret,
		Passphrase:  "passphrase",
		HTTPTimeout: 100 * time.Millisecond,
	})
	_, err := client.Balances(context.Background())
	if err == nil {
		t.Fatal("Balances() error = nil, want network error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("network error exposes secret: %v", err)
	}
}

func TestCalculateWeightedAvgPrice(t *testing.T) {
	if got := calculateWeightedAvgPrice("100", "100500"); got != "1005" {
		t.Fatalf("weighted average = %q, want 1005", got)
	}
	if got := calculateWeightedAvgPrice("0", "100500"); got != "" {
		t.Fatalf("zero-volume weighted average = %q, want empty", got)
	}
}

func TestSignUsesBase64HMACSHA256(t *testing.T) {
	if got := sign("1700000000000GET/api/v3/account", "test-secret-key"); got == "" || bytes.Contains([]byte(got), []byte("-")) {
		t.Fatalf("signature = %q, want non-empty standard base64", got)
	}
}
