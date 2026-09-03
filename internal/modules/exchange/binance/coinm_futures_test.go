package binance

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

func TestClientCOINMFuturesPublicEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-MBX-APIKEY"); got != "" {
			t.Errorf("public COIN-M Futures request API key header = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/dapi/v1/ping":
			if r.URL.RawQuery != "" {
				t.Errorf("COIN-M Futures ping query = %q, want empty", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{}`))
		case "/dapi/v1/time":
			_, _ = w.Write([]byte(`{"serverTime":1700000000123}`))
		case "/dapi/v1/exchangeInfo":
			_, _ = w.Write([]byte(`{"timezone":"UTC","serverTime":1700000000000,"rateLimits":[{"rateLimitType":"REQUEST_WEIGHT","interval":"MINUTE","intervalNum":1,"limit":2400}],"exchangeFilters":[],"symbols":[{"symbol":"BTCUSD_PERP","pair":"BTCUSD","contractType":"PERPETUAL","deliveryDate":4133404800000,"onboardDate":1597042800000,"contractStatus":"TRADING","maintMarginPercent":"2.5000","requiredMarginPercent":"5.0000","baseAsset":"BTC","quoteAsset":"USD","marginAsset":"BTC","pricePrecision":1,"quantityPrecision":0,"baseAssetPrecision":8,"quotePrecision":8,"underlyingType":"COIN","underlyingSubType":["PoW"],"equalQtyPrecision":4,"triggerProtect":"0.0500","liquidationFee":"0.015000","marketTakeBound":"0.05","maxMoveOrderLimit":10000,"contractSize":100,"filters":[],"orderTypes":["LIMIT","MARKET"],"timeInForce":["GTC"],"permissionSets":["GRID"]}]}`))
		case "/dapi/v1/depth":
			if got := r.URL.Query().Get("symbol"); got != "BTCUSD_PERP" {
				t.Errorf("COIN-M Futures depth symbol = %q, want BTCUSD_PERP", got)
			}
			if got := r.URL.Query().Get("limit"); got != "20" {
				t.Errorf("COIN-M Futures depth limit = %q, want 20", got)
			}
			_, _ = w.Write([]byte(`{"lastUpdateId":123,"symbol":"BTCUSD_PERP","pair":"BTCUSD","E":1700000000100,"T":1700000000099,"bids":[["100.00","1.50"]],"asks":[["100.10","2.00"]]}`))
		case "/dapi/v1/klines":
			query := r.URL.Query()
			if got := query.Get("symbol"); got != "BTCUSD_PERP" {
				t.Errorf("COIN-M Futures klines symbol = %q, want BTCUSD_PERP", got)
			}
			if got := query.Get("interval"); got != "1m" {
				t.Errorf("COIN-M Futures klines interval = %q, want 1m", got)
			}
			if got := query.Get("startTime"); got != "1700000000000" {
				t.Errorf("COIN-M Futures klines startTime = %q, want 1700000000000", got)
			}
			if got := query.Get("endTime"); got != "1700000060000" {
				t.Errorf("COIN-M Futures klines endTime = %q, want 1700000060000", got)
			}
			if got := query.Get("timeZone"); got != "" {
				t.Errorf("COIN-M Futures klines timeZone = %q, want empty", got)
			}
			_, _ = w.Write([]byte(`[[1700000000000,"100.00","101.00","99.00","100.50","10",1700000059999,"1005.00",42,"5","502.50","0"]]`))
		case "/dapi/v1/ticker/24hr":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSD_PERP","pair":"BTCUSD","priceChange":"10.00","priceChangePercent":"1.00","weightedAvgPrice":"1005.00","lastPrice":"1000.00","lastQty":"0.10","openPrice":"990.00","highPrice":"1010.00","lowPrice":"980.00","volume":"100.00","baseVolume":"10.00","openTime":1700000000000,"closeTime":1700086399999,"firstId":1,"lastId":100,"count":100}]`))
		case "/dapi/v1/ticker/price":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSD_PERP","ps":"BTCUSD","price":"1000.00","time":1700000000123}]`))
		case "/dapi/v1/ticker/bookTicker":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSD_PERP","pair":"BTCUSD","bidPrice":"999.90","bidQty":"1","askPrice":"1000.10","askQty":"2","time":1700000000123,"lastUpdateId":123}]`))
		case "/dapi/v1/premiumIndex":
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSD_PERP","pair":"BTCUSD","markPrice":"1000.00","indexPrice":"999.90","estimatedSettlePrice":"1000.10","lastFundingRate":"0.00010000","interestRate":"0.00010000","nextFundingTime":1700028800000,"time":1700000000123}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BinanceConfig{
		BaseURL:             "https://spot.invalid",
		FuturesBaseURL:      "https://usdm.invalid",
		CoinMFuturesBaseURL: server.URL,
		HTTPTimeout:         time.Second,
	})
	if err := client.CoinMFuturesPing(context.Background()); err != nil {
		t.Fatalf("CoinMFuturesPing() error = %v", err)
	}
	serverTime, err := client.CoinMFuturesServerTime(context.Background())
	if err != nil || serverTime.ServerTime != 1700000000123 {
		t.Fatalf("CoinMFuturesServerTime() = %#v, error = %v", serverTime, err)
	}

	info, err := client.CoinMFuturesExchangeInfo(context.Background())
	if err != nil {
		t.Fatalf("CoinMFuturesExchangeInfo() error = %v", err)
	}
	if info.Timezone != "UTC" || len(info.RateLimits) != 1 || len(info.Symbols) != 1 {
		t.Fatalf("COIN-M Futures exchange info = %#v", info)
	}
	if symbol := info.Symbols[0]; symbol.ContractStatus != "TRADING" || symbol.ContractSize != 100 || symbol.MarginAsset != "BTC" {
		t.Fatalf("COIN-M Futures symbol = %#v", symbol)
	}

	orderBook, err := client.CoinMFuturesDepth(context.Background(), "btcusd_perp", 20)
	if err != nil {
		t.Fatalf("CoinMFuturesDepth() error = %v", err)
	}
	if orderBook.LastUpdateID != 123 || orderBook.Pair != "BTCUSD" || !reflect.DeepEqual(orderBook.Asks, [][]string{{"100.10", "2.00"}}) {
		t.Fatalf("COIN-M Futures order book = %#v", orderBook)
	}

	startTime := int64(1700000000000)
	endTime := int64(1700000060000)
	klines, err := client.CoinMFuturesKlines(context.Background(), exchange.KlinesRequest{
		Symbol:    "BTCUSD_PERP",
		Interval:  "1m",
		StartTime: &startTime,
		EndTime:   &endTime,
		TimeZone:  "UTC+8",
		Limit:     1,
	})
	if err != nil || len(klines) != 1 || klines[0].Close != "100.50" || klines[0].NumberOfTrades != 42 {
		t.Fatalf("CoinMFuturesKlines() = %#v, error = %v", klines, err)
	}

	ticker, err := client.CoinMFuturesTicker24hr(context.Background(), "BTCUSD_PERP")
	if err != nil || ticker.Pair != "BTCUSD" || ticker.BaseVolume != "10.00" {
		t.Fatalf("CoinMFuturesTicker24hr() = %#v, error = %v", ticker, err)
	}
	price, err := client.CoinMFuturesTickerPrice(context.Background(), "BTCUSD_PERP")
	if err != nil || price.Pair != "BTCUSD" || price.Price != "1000.00" || price.Time != 1700000000123 {
		t.Fatalf("CoinMFuturesTickerPrice() = %#v, error = %v", price, err)
	}
	bookTicker, err := client.CoinMFuturesBookTicker(context.Background(), "BTCUSD_PERP")
	if err != nil || bookTicker.LastUpdateID != 123 || bookTicker.AskPrice != "1000.10" {
		t.Fatalf("CoinMFuturesBookTicker() = %#v, error = %v", bookTicker, err)
	}
	premium, err := client.CoinMFuturesPremiumIndex(context.Background(), "BTCUSD_PERP")
	if err != nil || premium.Pair != "BTCUSD" || premium.MarkPrice != "1000.00" || premium.NextFundingTime != 1700028800000 {
		t.Fatalf("CoinMFuturesPremiumIndex() = %#v, error = %v", premium, err)
	}
}

func TestClientCOINMFuturesValidation(t *testing.T) {
	client := New(config.BinanceConfig{BaseURL: "https://example.com", CoinMFuturesBaseURL: "https://example.com", HTTPTimeout: time.Second})

	_, err := client.CoinMFuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSD_PERP", Interval: "1s"})
	var parameterError exchange.InvalidParameterError
	if !errors.As(err, &parameterError) || parameterError.Parameter != "interval" {
		t.Fatalf("CoinMFuturesKlines() error = %v, want interval validation error", err)
	}

	_, err = client.CoinMFuturesDepth(context.Background(), "BTCUSD_PERP", 30)
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("CoinMFuturesDepth() error = %v, want limit validation error", err)
	}
}

func TestClientCOINMFuturesEmptyArrayResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := New(config.BinanceConfig{CoinMFuturesBaseURL: server.URL, HTTPTimeout: time.Second})
	_, err := client.CoinMFuturesTickerPrice(context.Background(), "BTCUSD_PERP")
	if err == nil || err.Error() != "decode COIN-M Futures price ticker response: empty response" {
		t.Fatalf("CoinMFuturesTickerPrice() error = %v, want empty response error", err)
	}
}
