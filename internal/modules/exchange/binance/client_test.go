package binance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

func TestClientSpotPublicEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-MBX-APIKEY"); got != "" {
			t.Errorf("public request API key header = %q, want empty", got)
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
			_, _ = w.Write([]byte(`{"timezone":"UTC","serverTime":1700000000000,"rateLimits":[{"rateLimitType":"REQUEST_WEIGHT","interval":"MINUTE","intervalNum":1,"limit":6000}],"exchangeFilters":[],"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","baseAssetPrecision":8,"quoteAsset":"USDT","quotePrecision":8,"quoteAssetPrecision":8,"orderTypes":["LIMIT"],"isSpotTradingAllowed":true,"filters":[{"filterType":"PRICE_FILTER","tickSize":"0.01000000"}]}]}`))
		case "/api/v3/depth":
			query := r.URL.Query()
			if got := query.Get("symbol"); got != "BTCUSDT" {
				t.Errorf("depth symbol = %q, want BTCUSDT", got)
			}
			if got := query.Get("limit"); got != "20" {
				t.Errorf("depth limit = %q, want 20", got)
			}
			_, _ = w.Write([]byte(`{"lastUpdateId":123,"bids":[["100.00","1.50"]],"asks":[["100.10","2.00"]]}`))
		case "/api/v3/klines":
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
			if got := query.Get("timeZone"); got != "UTC+8" {
				t.Errorf("klines timeZone = %q, want UTC+8", got)
			}
			_, _ = w.Write([]byte(`[[1700000000000,"100.00","101.00","99.00","100.50","10.00",1700000059999,"1005.00",42,"5.00","502.50","0"]]`))
		case "/api/v3/ticker/24hr":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","priceChange":"10.00","priceChangePercent":"1.00","weightedAvgPrice":"1005.00","prevClosePrice":"990.00","lastPrice":"1000.00","lastQty":"0.10","bidPrice":"999.90","bidQty":"1.00","askPrice":"1000.10","askQty":"2.00","openPrice":"990.00","highPrice":"1010.00","lowPrice":"980.00","volume":"100.00","quoteVolume":"100500.00","openTime":1700000000000,"closeTime":1700086399999,"firstId":1,"lastId":100,"count":100}`))
		case "/api/v3/ticker/price":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","price":"1000.00"}`))
		case "/api/v3/ticker/bookTicker":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","bidPrice":"999.90","bidQty":"1.00","askPrice":"1000.10","askQty":"2.00"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BinanceConfig{BaseURL: server.URL, HTTPTimeout: time.Second})
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
	if got := info.RateLimits[0].Limit; got != 6000 {
		t.Fatalf("rate limit = %d, want 6000", got)
	}

	orderBook, err := client.Depth(context.Background(), "btcusdt", 20)
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
	if len(klines) != 1 || klines[0].Open != "100.00" || klines[0].NumberOfTrades != 42 {
		t.Fatalf("klines = %#v", klines)
	}

	ticker, err := client.Ticker24hr(context.Background(), "BTCUSDT")
	if err != nil || ticker.LastPrice != "1000.00" {
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
	client := New(config.BinanceConfig{BaseURL: "https://example.com", HTTPTimeout: time.Second})

	_, err := client.Depth(context.Background(), "", 100)
	var parameterError exchange.InvalidParameterError
	if !errors.As(err, &parameterError) || parameterError.Parameter != "symbol" {
		t.Fatalf("Depth() error = %v, want symbol validation error", err)
	}

	_, err = client.Klines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "bad"})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "interval" {
		t.Fatalf("Klines() error = %v, want interval validation error", err)
	}

	_, err = client.Klines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", Limit: 1001})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("Klines() error = %v, want limit validation error", err)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":-1003,"msg":"Too many requests"}`))
	}))
	defer server.Close()

	client := New(config.BinanceConfig{BaseURL: server.URL, HTTPTimeout: time.Second})
	err := client.Ping(context.Background())
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("Ping() error = %v, want APIError", err)
	}
	if apiError.HTTPStatus != http.StatusTooManyRequests || apiError.Code != -1003 {
		t.Fatalf("API error = %#v", apiError)
	}
}

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
