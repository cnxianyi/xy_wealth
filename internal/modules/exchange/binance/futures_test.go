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

func TestClientUSDSMFuturesPublicEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-MBX-APIKEY"); got != "" {
			t.Errorf("public Futures request API key header = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/fapi/v1/ping":
			_, _ = w.Write([]byte(`{}`))
		case "/fapi/v1/time":
			_, _ = w.Write([]byte(`{"serverTime":1700000000123}`))
		case "/fapi/v1/exchangeInfo":
			_, _ = w.Write([]byte(`{"timezone":"UTC","serverTime":1700000000000,"futuresType":"U_MARGINED","rateLimits":[{"rateLimitType":"REQUEST_WEIGHT","interval":"MINUTE","intervalNum":1,"limit":2400}],"exchangeFilters":[],"assets":[{"asset":"USDT","marginAvailable":true,"autoAssetExchange":"-10000"}],"symbols":[{"symbol":"BTCUSDT","pair":"BTCUSDT","contractType":"PERPETUAL","deliveryDate":4133404800000,"onboardDate":1567965300000,"status":"TRADING","maintMarginPercent":"2.5000","requiredMarginPercent":"5.0000","baseAsset":"BTC","quoteAsset":"USDT","marginAsset":"USDT","pricePrecision":2,"quantityPrecision":3,"baseAssetPrecision":8,"quotePrecision":8,"underlyingType":"COIN","underlyingSubType":["PoW"],"triggerProtect":"0.0500","liquidationFee":"0.012500","marketTakeBound":"0.05","maxMoveOrderLimit":10000,"filters":[],"orderTypes":["LIMIT","MARKET"],"timeInForce":["GTC"],"permissionSets":["GRID"]}]}`))
		case "/fapi/v1/depth":
			if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
				t.Errorf("Futures depth symbol = %q, want BTCUSDT", got)
			}
			if got := r.URL.Query().Get("limit"); got != "20" {
				t.Errorf("Futures depth limit = %q, want 20", got)
			}
			_, _ = w.Write([]byte(`{"lastUpdateId":123,"symbol":"BTCUSDT","pair":"BTCUSDT","E":1700000000100,"T":1700000000099,"bids":[["100.00","1.50"]],"asks":[["100.10","2.00"]]}`))
		case "/fapi/v1/klines":
			if got := r.URL.Query().Get("interval"); got != "1m" {
				t.Errorf("Futures klines interval = %q, want 1m", got)
			}
			_, _ = w.Write([]byte(`[[1700000000000,"100.00","101.00","99.00","100.50","10.00",1700000059999,"1005.00",42,"5.00","502.50","0"]]`))
		case "/fapi/v1/ticker/24hr":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","priceChange":"10.00","priceChangePercent":"1.00","weightedAvgPrice":"1005.00","lastPrice":"1000.00","lastQty":"0.10","openPrice":"990.00","highPrice":"1010.00","lowPrice":"980.00","volume":"100.00","baseVolume":"10.00","quoteVolume":"100500.00","openTime":1700000000000,"closeTime":1700086399999,"firstId":1,"lastId":100,"count":100}`))
		case "/fapi/v1/ticker/price":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","price":"1000.00"}`))
		case "/fapi/v1/ticker/bookTicker":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","bidPrice":"999.90","bidQty":"1.00","askPrice":"1000.10","askQty":"2.00"}`))
		case "/fapi/v1/premiumIndex":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","markPrice":"1000.00","indexPrice":"999.90","estimatedSettlePrice":"1000.10","lastFundingRate":"0.00010000","interestRate":"0.00010000","nextFundingTime":1700028800000,"time":1700000000123}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BinanceConfig{BaseURL: "https://spot.invalid", FuturesBaseURL: server.URL, HTTPTimeout: time.Second})
	if err := client.FuturesPing(context.Background()); err != nil {
		t.Fatalf("FuturesPing() error = %v", err)
	}
	serverTime, err := client.FuturesServerTime(context.Background())
	if err != nil || serverTime.ServerTime != 1700000000123 {
		t.Fatalf("FuturesServerTime() = %#v, error = %v", serverTime, err)
	}

	info, err := client.FuturesExchangeInfo(context.Background())
	if err != nil {
		t.Fatalf("FuturesExchangeInfo() error = %v", err)
	}
	if info.Timezone != "UTC" || info.FuturesType != "U_MARGINED" || len(info.Assets) != 1 || len(info.Symbols) != 1 {
		t.Fatalf("FuturesExchangeInfo() = %#v", info)
	}
	if info.Symbols[0].ContractType != "PERPETUAL" || info.Symbols[0].MarginAsset != "USDT" || len(info.Symbols[0].PermissionSets) != 1 {
		t.Fatalf("Futures symbol = %#v", info.Symbols[0])
	}

	orderBook, err := client.FuturesDepth(context.Background(), "btcusdt", 20)
	if err != nil {
		t.Fatalf("FuturesDepth() error = %v", err)
	}
	if orderBook.LastUpdateID != 123 || orderBook.EventTime != 1700000000100 || !reflect.DeepEqual(orderBook.Asks, [][]string{{"100.10", "2.00"}}) {
		t.Fatalf("Futures order book = %#v", orderBook)
	}

	klines, err := client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", Limit: 1})
	if err != nil || len(klines) != 1 || klines[0].Close != "100.50" {
		t.Fatalf("FuturesKlines() = %#v, error = %v", klines, err)
	}
	ticker, err := client.FuturesTicker24hr(context.Background(), "BTCUSDT")
	if err != nil || ticker.LastPrice != "1000.00" || ticker.BaseVolume != "10.00" {
		t.Fatalf("FuturesTicker24hr() = %#v, error = %v", ticker, err)
	}
	price, err := client.FuturesTickerPrice(context.Background(), "BTCUSDT")
	if err != nil || price.Price != "1000.00" {
		t.Fatalf("FuturesTickerPrice() = %#v, error = %v", price, err)
	}
	bookTicker, err := client.FuturesBookTicker(context.Background(), "BTCUSDT")
	if err != nil || bookTicker.BidPrice != "999.90" {
		t.Fatalf("FuturesBookTicker() = %#v, error = %v", bookTicker, err)
	}
	premium, err := client.FuturesPremiumIndex(context.Background(), "BTCUSDT")
	if err != nil || premium.MarkPrice != "1000.00" || premium.NextFundingTime != 1700028800000 {
		t.Fatalf("FuturesPremiumIndex() = %#v, error = %v", premium, err)
	}
}

func TestClientUSDSMFuturesValidation(t *testing.T) {
	client := New(config.BinanceConfig{BaseURL: "https://example.com", FuturesBaseURL: "https://example.com", HTTPTimeout: time.Second})

	_, err := client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1s"})
	var parameterError exchange.InvalidParameterError
	if !errors.As(err, &parameterError) || parameterError.Parameter != "interval" {
		t.Fatalf("FuturesKlines() error = %v, want interval validation error", err)
	}

	_, err = client.FuturesDepth(context.Background(), "BTCUSDT", 1001)
	if !errors.As(err, &parameterError) || parameterError.Parameter != "limit" {
		t.Fatalf("FuturesDepth() error = %v, want limit validation error", err)
	}
}
