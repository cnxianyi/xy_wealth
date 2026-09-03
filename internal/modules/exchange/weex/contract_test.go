package weex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

func TestClientContractPublicEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; len(got) < len(weexContractPathPrefix) || got[:len(weexContractPathPrefix)] != weexContractPathPrefix {
			t.Errorf("path = %q, want Contract V3 prefix", got)
		}
		if got := r.Header.Get(accessKeyHeader); got != "" {
			t.Errorf("public request API key header = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/capi/v3/market/time":
			_, _ = w.Write([]byte(`{"serverTime":1700000000123}`))
		case "/capi/v3/market/exchangeInfo":
			_, _ = w.Write([]byte(`{"assets":[{"asset":"USDT","marginAvailable":true}],"rateLimits":[{"rateLimitType":"REQUEST_WEIGHT","interval":"SECOND","intervalNum":10,"limit":2000}],"symbols":[{"symbol":"BTCUSDT","displaySymbol":"BTCUSDT","baseAsset":"BTC","quoteAsset":"USDT","marginAsset":"USDT","contractType":"PERPETUAL","underlyingType":"COIN","underlyingSubType":["PoW"],"pricePrecision":1,"quantityPrecision":6,"baseAssetPrecision":3,"quotePrecision":8,"contractVal":0.000001,"delivery":["00:00:00","06:00:00"],"forwardContractFlag":true,"minLeverage":1,"maxLeverage":408,"buyLimitPriceRatio":0.016,"sellLimitPriceRatio":0.015,"makerFeeRate":0.0002,"takerFeeRate":0.0008,"apiMakerFeeRate":0.0002,"apiTakerFeeRate":0.0008,"minOrderSize":0.0001,"maxOrderSize":10000,"maxPositionSize":20000000,"marketOpenLimitSize":12000}]}`))
		case "/capi/v3/market/depth":
			query := r.URL.Query()
			if got := query.Get("symbol"); got != "BTCUSDT" {
				t.Errorf("depth symbol = %q, want BTCUSDT", got)
			}
			if got := query.Get("limit"); got != "200" {
				t.Errorf("depth limit = %q, want 200", got)
			}
			_, _ = w.Write([]byte(`{"lastUpdateId":123,"bids":[["100.00","1.50"]],"asks":[["100.10","2.00"]]}`))
		case "/capi/v3/market/klines":
			query := r.URL.Query()
			if got := query.Get("symbol"); got != "ETHUSDT" {
				t.Errorf("klines symbol = %q, want ETHUSDT", got)
			}
			if got := query.Get("interval"); got != "1m" {
				t.Errorf("klines interval = %q, want 1m", got)
			}
			if got := query.Get("limit"); got != "1" {
				t.Errorf("klines limit = %q, want 1", got)
			}
			_, _ = w.Write([]byte(`[[1700000000000,"100.00","101.00","99.00","100.50","10.00",1700000059999,"1005.00",42,"5.00","502.50"]]`))
		case "/capi/v3/market/historyKlines":
			query := r.URL.Query()
			if got := query.Get("startTime"); got != "1700000000000" {
				t.Errorf("history klines startTime = %q, want 1700000000000", got)
			}
			if got := query.Get("endTime"); got != "1700000060000" {
				t.Errorf("history klines endTime = %q, want 1700000060000", got)
			}
			_, _ = w.Write([]byte(`[[1700000000000,"100.00","101.00","99.00","100.50","10.00",1700000059999,"1005.00",42,"5.00","502.50"]]`))
		case "/capi/v3/market/ticker/24hr":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","priceChange":"10.00","priceChangePercent":"1.00","lastPrice":"1000.00","openPrice":"990.00","highPrice":"1010.00","lowPrice":"980.00","volume":"100.00","quoteVolume":"100500.00","markPrice":"1000.10","indexPrice":"1000.00","openTime":1700000000000,"closeTime":1700086399999}]`))
		case "/capi/v3/market/symbolPrice":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","price":"1000.00","time":1700000000456}`))
		case "/capi/v3/market/ticker/bookTicker":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","bidPrice":"999.90","bidQty":"1.00","askPrice":"1000.10","askQty":"2.00","time":1700000000789}]`))
		case "/capi/v3/market/premiumIndex":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","markPrice":"1000.10","indexPrice":"1000.00","lastFundingRate":"0.00025","forecastFundingRate":"0.00030","interestRate":"0.0001","nextFundingTime":1700000800000,"time":1700000000999,"collectCycle":480}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.WeexConfig{ContractBaseURL: server.URL, HTTPTimeout: time.Second})
	if err := client.FuturesPing(context.Background()); err != nil {
		t.Fatalf("FuturesPing() error = %v", err)
	}
	serverTime, err := client.FuturesServerTime(context.Background())
	if err != nil {
		t.Fatalf("FuturesServerTime() error = %v", err)
	}
	if serverTime.ServerTime != 1700000000123 {
		t.Fatalf("server time = %d, want 1700000000123", serverTime.ServerTime)
	}

	info, err := client.FuturesExchangeInfo(context.Background())
	if err != nil {
		t.Fatalf("FuturesExchangeInfo() error = %v", err)
	}
	if info.Timezone != "UTC" || info.FuturesType != "WEEX_CONTRACT" || len(info.Assets) != 1 || len(info.Symbols) != 1 {
		t.Fatalf("exchange info = %#v", info)
	}
	if symbol := info.Symbols[0]; symbol.ContractVal != "0.000001" || symbol.MaxLeverage != 408 || symbol.Status != "TRADING" {
		t.Fatalf("contract symbol = %#v", symbol)
	}

	orderBook, err := client.FuturesDepth(context.Background(), "btcusdt", 200)
	if err != nil {
		t.Fatalf("FuturesDepth() error = %v", err)
	}
	if orderBook.Symbol != "BTCUSDT" || orderBook.LastUpdateID != 123 || !reflect.DeepEqual(orderBook.Bids, [][]string{{"100.00", "1.50"}}) {
		t.Fatalf("order book = %#v", orderBook)
	}

	klines, err := client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "ethusdt", Interval: "1m", Limit: 1})
	if err != nil {
		t.Fatalf("FuturesKlines() error = %v", err)
	}
	if len(klines) != 1 || klines[0].Close != "100.50" || klines[0].NumberOfTrades != 42 {
		t.Fatalf("klines = %#v", klines)
	}
	startTime := int64(1700000000000)
	endTime := int64(1700000060000)
	klines, err = client.FuturesKlines(context.Background(), exchange.KlinesRequest{
		Symbol:    "ethusdt",
		Interval:  "1m",
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     1,
	})
	if err != nil || len(klines) != 1 {
		t.Fatalf("FuturesKlines(history) = %#v, error = %v", klines, err)
	}

	ticker, err := client.FuturesTicker24hr(context.Background(), "BTCUSDT")
	if err != nil || ticker.WeightedAvgPrice != "1005" || ticker.MarkPrice != "1000.10" {
		t.Fatalf("FuturesTicker24hr() = %#v, error = %v", ticker, err)
	}
	price, err := client.FuturesTickerPrice(context.Background(), "BTCUSDT")
	if err != nil || price.Price != "1000.00" || price.Time != 1700000000456 {
		t.Fatalf("FuturesTickerPrice() = %#v, error = %v", price, err)
	}
	bookTicker, err := client.FuturesBookTicker(context.Background(), "BTCUSDT")
	if err != nil || bookTicker.AskPrice != "1000.10" || bookTicker.Time != 1700000000789 {
		t.Fatalf("FuturesBookTicker() = %#v, error = %v", bookTicker, err)
	}
	premium, err := client.FuturesPremiumIndex(context.Background(), "BTCUSDT")
	if err != nil || premium.ForecastFundingRate != "0.00030" || premium.CollectCycle != 480 {
		t.Fatalf("FuturesPremiumIndex() = %#v, error = %v", premium, err)
	}
}

func TestClientContractValidation(t *testing.T) {
	client := New(config.WeexConfig{ContractBaseURL: "https://example.com", HTTPTimeout: time.Second})
	var parameterError exchange.InvalidParameterError

	_, err := client.FuturesDepth(context.Background(), "BTCUSDT", 20)
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("FuturesDepth() error = %v, want limit validation error", err)
	}
	_, err = client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "2h"})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "interval" {
		t.Fatalf("FuturesKlines() error = %v, want interval validation error", err)
	}
	_, err = client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", Limit: 1001})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("FuturesKlines() error = %v, want limit validation error", err)
	}
	start := int64(1)
	_, err = client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", StartTime: &start, Limit: 101})
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("FuturesKlines(history) error = %v, want history limit validation error", err)
	}
}

func TestClientContractBalancesAndSignature(t *testing.T) {
	const (
		apiKey     = "test-api-key"
		secretKey  = "test-secret-key"
		passphrase = "test-passphrase"
		timestamp  = int64(1700000000000)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/capi/v3/account/balance" {
			t.Errorf("path = %q, want /capi/v3/account/balance", r.URL.Path)
		}
		if got := r.Header.Get(accessKeyHeader); got != apiKey {
			t.Errorf("API key header = %q, want %q", got, apiKey)
		}
		if got := r.Header.Get(accessTimestampHeader); got != "1700000000000" {
			t.Errorf("timestamp header = %q, want 1700000000000", got)
		}
		wantSignature := sign("1700000000000GET/capi/v3/account/balance", secretKey)
		if got := r.Header.Get(accessSignHeader); got != wantSignature {
			t.Errorf("signature = %q, want %q", got, wantSignature)
		}
		if got := r.Header.Get(accessPassphraseHeader); got != passphrase {
			t.Errorf("passphrase header = %q, want %q", got, passphrase)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"asset":"USDT","balance":"100.5","availableBalance":"90.5","frozen":"10","unrealizePnl":"0"},{"asset":"BTC","balance":"0","availableBalance":"0","frozen":"0","unrealizePnl":"0"}]`))
	}))
	defer server.Close()

	client := New(config.WeexConfig{ContractBaseURL: server.URL, APIKey: apiKey, SecretKey: secretKey, Passphrase: passphrase, HTTPTimeout: time.Second})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }
	balances, err := client.ContractBalances(context.Background())
	if err != nil {
		t.Fatalf("ContractBalances() error = %v", err)
	}
	if len(balances) != 1 || balances[0].Total != "100.5" || balances[0].Free != "90.5" || balances[0].Locked != "10" {
		t.Fatalf("contract balances = %#v", balances)
	}
}

func TestClientContractBalancesRequiresCredentials(t *testing.T) {
	client := New(config.WeexConfig{ContractBaseURL: "https://example.com", HTTPTimeout: time.Second})
	_, err := client.ContractBalances(context.Background())
	if !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("ContractBalances() error = %v, want ErrCredentialsMissing", err)
	}
}

func TestClientContractPositionsAndSignature(t *testing.T) {
	const (
		apiKey     = "test-api-key"
		secretKey  = "test-secret-key"
		passphrase = "test-passphrase"
		timestamp  = int64(1700000000000)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/capi/v3/account/position/allPosition"
		wantPayload := "1700000000000GET/capi/v3/account/position/allPosition"
		if r.URL.Path == "/capi/v3/account/position/singlePosition" {
			wantPath = "/capi/v3/account/position/singlePosition"
			if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
				t.Errorf("position symbol = %q, want BTCUSDT", got)
			}
			wantPayload = "1700000000000GET" + wantPath + "?symbol=BTCUSDT"
		} else if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
		}
		if got := r.Header.Get(accessKeyHeader); got != apiKey {
			t.Errorf("API key header = %q, want %q", got, apiKey)
		}
		if got := r.Header.Get(accessTimestampHeader); got != "1700000000000" {
			t.Errorf("timestamp header = %q, want 1700000000000", got)
		}
		if got, want := r.Header.Get(accessSignHeader), sign(wantPayload, secretKey); got != want {
			t.Errorf("signature = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"asset":"USDT","symbol":"BTCUSDT","side":"LONG","marginType":"CROSSED","separatedMode":"COMBINED","separatedOpenOrderId":0,"leverage":"20","size":"0.020000","openValue":"1801.0670000","openFee":"0.70731060","fundingFee":"1.22618160","marginSize":"17.154980","isolatedMargin":"0","isAutoAppendIsolatedMargin":false,"cumOpenSize":"0.020000","cumOpenValue":"1801.0670000","cumOpenFee":"0.70731060","cumCloseSize":"0","cumCloseValue":"0","cumCloseFee":"0","cumFundingFee":"1.22618160","cumLiquidateFee":"0","createdMatchSequenceId":10,"updatedMatchSequenceId":11,"createdTime":1700000000000,"updatedTime":1700000001000,"unrealizePnl":"-85.5690000","liquidatePrice":"0"},{"id":2,"asset":"USDT","symbol":"UNIUSDT","side":"SHORT","marginType":"CROSSED","separatedMode":"COMBINED","leverage":"20","size":"0.00000000"}]`))
	}))
	defer server.Close()

	client := New(config.WeexConfig{ContractBaseURL: server.URL, APIKey: apiKey, SecretKey: secretKey, Passphrase: passphrase, HTTPTimeout: time.Second})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	positions, err := client.ContractPositions(context.Background(), "")
	if err != nil {
		t.Fatalf("ContractPositions(all) error = %v", err)
	}
	if len(positions) != 1 || positions[0].Side != "LONG" || positions[0].Size != "0.020000" || positions[0].UnrealizePnl != "-85.5690000" {
		t.Fatalf("all positions = %#v", positions)
	}

	positions, err = client.ContractPositions(context.Background(), "btcusdt")
	if err != nil {
		t.Fatalf("ContractPositions(single) error = %v", err)
	}
	if len(positions) != 1 || positions[0].Symbol != "BTCUSDT" {
		t.Fatalf("single positions = %#v", positions)
	}
}

func TestClientContractPositionsRequiresCredentials(t *testing.T) {
	client := New(config.WeexConfig{ContractBaseURL: "https://example.com", HTTPTimeout: time.Second})
	_, err := client.ContractPositions(context.Background(), "BTCUSDT")
	if !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("ContractPositions() error = %v, want ErrCredentialsMissing", err)
	}
}
