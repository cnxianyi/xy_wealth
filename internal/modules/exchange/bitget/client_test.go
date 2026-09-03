package bitget

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

func TestNewUsesBitgetDefaults(t *testing.T) {
	client := New(config.BitgetConfig{})
	if client.Name() != "bitget" {
		t.Fatalf("Name() = %q, want bitget", client.Name())
	}
	if client.baseURL != "https://api.bitget.com" {
		t.Fatalf("base URL = %q, want https://api.bitget.com", client.baseURL)
	}
	if client.httpClient.Timeout != 10*time.Second {
		t.Fatalf("HTTP timeout = %s, want 10s", client.httpClient.Timeout)
	}
	var provider exchange.SpotProvider = client
	if provider.Name() != "bitget" {
		t.Fatalf("provider name = %q, want bitget", provider.Name())
	}
}

func TestClientSpotEndpointsAndEnvelope(t *testing.T) {
	const (
		apiKey     = "bitget-api-key"
		secretKey  = "bitget-secret-key"
		passphrase = "bitget-passphrase"
		timestamp  = int64(1700000000000)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/spot/account/assets" {
			if got := r.Header.Get(accessKeyHeader); got != "" {
				t.Errorf("public request API key header = %q, want empty", got)
			}
		}
		if got := r.URL.Path; !strings.HasPrefix(got, "/api/v2/") {
			t.Errorf("path = %q, want /api/v2 prefix", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/public/time":
			_, _ = w.Write([]byte(`{"code":"00000","message":"success","data":{"serverTime":"1700000000123"}}`))
		case "/api/v2/spot/public/symbols":
			if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
				t.Errorf("symbols symbol = %q, want BTCUSDT", got)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSDT","baseCoin":"BTC","quoteCoin":"USDT","minTradeAmount":"0.0001","maxTradeAmount":"900000","takerFeeRate":"0.001","makerFeeRate":"0.0005","pricePrecision":"2","quantityPrecision":"6","quotePrecision":"8","status":"online","minTradeUSDT":"1","buyLimitPriceRatio":"0.05","sellLimitPriceRatio":"0.05"}]}`))
		case "/api/v2/spot/market/orderbook":
			query := r.URL.Query()
			if query.Get("symbol") != "BTCUSDT" || query.Get("type") != "step0" || query.Get("limit") != "20" {
				t.Errorf("order book query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"asks":[["101.00","2"]],"bids":[["100.00","3"]],"ts":"1700000000456"}}`))
		case "/api/v2/spot/market/candles":
			query := r.URL.Query()
			if query.Get("granularity") != "1min" || query.Get("limit") != "2" {
				t.Errorf("candles query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[["1700000000000","100","110","90","105","2","200","200"],["1699999940000","99","101","98","100","1","100","100"]]}`))
		case "/api/v2/spot/market/tickers":
			if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
				t.Errorf("tickers symbol = %q, want BTCUSDT", got)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSDT","high24h":"110","open":"100","low24h":"90","lastPr":"105","quoteVolume":"200","baseVolume":"2","bidPr":"104.9","askPr":"105.1","bidSz":"1.5","askSz":"1.2","ts":"1700000000789","change24h":"0.05"}]}`))
		case "/api/v2/spot/account/assets":
			if got := r.Header.Get(accessKeyHeader); got != apiKey {
				t.Errorf("signed API key header = %q, want %q", got, apiKey)
			}
			if got := r.Header.Get(accessPassphraseHeader); got != passphrase {
				t.Errorf("passphrase header = %q, want %q", got, passphrase)
			}
			if got := r.Header.Get(accessTimestampHeader); got != "1700000000000" {
				t.Errorf("timestamp header = %q", got)
			}
			payload := r.Header.Get(accessTimestampHeader) + http.MethodGet + r.URL.Path + "?" + r.URL.RawQuery
			if got, want := r.Header.Get(accessSignHeader), sign(payload, secretKey); got != want {
				t.Errorf("signature = %q, want %q", got, want)
			}
			if got := r.URL.Query().Get("assetType"); got != "all" {
				t.Errorf("assetType = %q, want all", got)
			}
			_, _ = w.Write([]byte(`{"code":"00000","message":"success","data":[{"coin":"usdt","available":"10","frozen":"1","locked":"2","limitAvailable":"0","uTime":"1700000000"},{"coin":"eth","available":"0","frozen":"0","locked":"0","limitAvailable":"0","uTime":"1700000000"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BitgetConfig{BaseURL: server.URL, APIKey: apiKey, SecretKey: secretKey, Passphrase: passphrase, IncludeZero: false, HTTPTimeout: time.Second})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	serverTime, err := client.ServerTime(context.Background())
	if err != nil || serverTime.ServerTime != 1700000000123 {
		t.Fatalf("ServerTime() = %#v, error = %v", serverTime, err)
	}
	info, err := client.ExchangeInfo(context.Background(), "btcusdt")
	if err != nil || len(info.Symbols) != 1 || info.Symbols[0].BaseAssetPrecision != 6 || info.Symbols[0].MinTradeUSDT != "1" {
		t.Fatalf("ExchangeInfo() = %#v, error = %v", info, err)
	}
	orderBook, err := client.Depth(context.Background(), "btcusdt", 20)
	if err != nil || orderBook.Time != 1700000000456 || orderBook.Bids[0][0] != "100.00" {
		t.Fatalf("Depth() = %#v, error = %v", orderBook, err)
	}
	klines, err := client.Klines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", Limit: 2})
	if err != nil || len(klines) != 2 || klines[0].QuoteAssetVolume != "200" {
		t.Fatalf("Klines() = %#v, error = %v", klines, err)
	}
	ticker, err := client.Ticker24hr(context.Background(), "BTCUSDT")
	if err != nil || ticker.PriceChange != "5" || ticker.PriceChangePercent != "5" || ticker.WeightedAvgPrice != "100" {
		t.Fatalf("Ticker24hr() = %#v, error = %v", ticker, err)
	}
	price, err := client.TickerPrice(context.Background(), "BTCUSDT")
	if err != nil || price.Price != "105" || price.Time != 1700000000789 {
		t.Fatalf("TickerPrice() = %#v, error = %v", price, err)
	}
	bookTicker, err := client.BookTicker(context.Background(), "BTCUSDT")
	if err != nil || bookTicker.BidPrice != "104.9" || bookTicker.AskQty != "1.2" {
		t.Fatalf("BookTicker() = %#v, error = %v", bookTicker, err)
	}
	balances, err := client.Balances(context.Background())
	if err != nil || len(balances) != 1 || balances[0].Symbol != "USDT" || balances[0].Locked != "3" || balances[0].Total != "13" {
		t.Fatalf("Balances() = %#v, error = %v", balances, err)
	}
}

func TestClientValidationAndErrors(t *testing.T) {
	client := New(config.BitgetConfig{BaseURL: "https://example.com", HTTPTimeout: time.Second})
	if _, err := client.Depth(context.Background(), "", 100); err == nil {
		t.Fatal("Depth() error = nil, want symbol validation error")
	}
	if _, err := client.Depth(context.Background(), "BTCUSDT", 151); err == nil {
		t.Fatal("Depth() error = nil, want limit validation error")
	}
	if _, err := client.Klines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "bad"}); err == nil {
		t.Fatal("Klines() error = nil, want interval validation error")
	}
	if _, err := client.Balances(context.Background()); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("Balances() error = %v, want ErrCredentialsMissing", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"43001","msg":"invalid request"}`))
	}))
	defer server.Close()
	client = New(config.BitgetConfig{BaseURL: server.URL, HTTPTimeout: time.Second})
	err := client.Ping(context.Background())
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.Code != "43001" {
		t.Fatalf("Ping() error = %v, want Bitget APIError", err)
	}
}
