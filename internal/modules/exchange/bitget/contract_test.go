package bitget

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

func TestClientUSDTFuturesEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(accessKeyHeader); got != "" {
			t.Errorf("public request API key header = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/public/time":
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"serverTime":"1700000000123"}}`))
		case "/api/v2/mix/market/contracts":
			if got := r.URL.Query().Get("productType"); got != bitgetUSDTFuturesProductType {
				t.Errorf("contract productType = %q, want %s", got, bitgetUSDTFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSDT","baseCoin":"BTC","quoteCoin":"USDT","buyLimitPriceRatio":"0.05","sellLimitPriceRatio":"0.05","makerFeeRate":"0.0002","takerFeeRate":"0.0006","supportMarginCoins":["USDT"],"minTradeNum":"0.0001","volumePlace":"4","pricePlace":"1","sizeMultiplier":"0.0001","minTradeUSDT":"5","maxPositionNum":"200","symbolType":"perpetual","symbolStatus":"normal","deliveryTime":"","launchTime":"","minLever":"1","maxLever":"150","maxMarketOrderQty":"220","maxOrderQty":"1200"}]}`))
		case "/api/v2/mix/market/merge-depth":
			query := r.URL.Query()
			if query.Get("productType") != bitgetUSDTFuturesProductType || query.Get("symbol") != "BTCUSDT" || query.Get("limit") != "5" {
				t.Errorf("futures depth query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"asks":[[101.2,2.5]],"bids":[[100.8,3]],"ts":"1700000000456"}}`))
		case "/api/v2/mix/market/candles":
			query := r.URL.Query()
			if query.Get("productType") != bitgetUSDTFuturesProductType || query.Get("granularity") != "1m" {
				t.Errorf("futures candles query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[["1700000000000","100","110","90","105","2","200"]]}`))
		case "/api/v2/mix/market/ticker":
			query := r.URL.Query()
			if query.Get("productType") != bitgetUSDTFuturesProductType || query.Get("symbol") != "BTCUSDT" {
				t.Errorf("futures ticker query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSDT","lastPr":"105","askPr":"105.1","bidPr":"104.9","bidSz":"1.5","askSz":"1.2","high24h":"110","low24h":"90","ts":"1700000000789","change24h":"0.05","baseVolume":"2","quoteVolume":"200","open24h":"100","indexPrice":"104.8","fundingRate":"0.0001","markPrice":"105.0"}]}`))
		case "/api/v2/mix/market/symbol-price":
			if r.URL.Query().Get("productType") != bitgetUSDTFuturesProductType {
				t.Errorf("futures price productType = %q", r.URL.Query().Get("productType"))
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSDT","price":"105","indexPrice":"104.8","markPrice":"105.0","ts":"1700000000789"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BitgetConfig{BaseURL: server.URL, HTTPTimeout: time.Second})
	if err := client.FuturesPing(context.Background()); err != nil {
		t.Fatalf("FuturesPing() error = %v", err)
	}
	serverTime, err := client.FuturesServerTime(context.Background())
	if err != nil || serverTime.ServerTime != 1700000000123 {
		t.Fatalf("FuturesServerTime() = %#v, error = %v", serverTime, err)
	}
	info, err := client.FuturesExchangeInfo(context.Background())
	if err != nil || len(info.Symbols) != 1 || info.Symbols[0].ContractType != "PERPETUAL" || info.Symbols[0].Status != "TRADING" || info.Symbols[0].MaxLeverage != 150 || info.Symbols[0].MinTradeUSDT != "5" {
		t.Fatalf("FuturesExchangeInfo() = %#v, error = %v", info, err)
	}
	depth, err := client.FuturesDepth(context.Background(), "btcusdt", 5)
	if err != nil || depth.EventTime != 1700000000456 || depth.Bids[0][0] != "100.8" || depth.Asks[0][1] != "2.5" {
		t.Fatalf("FuturesDepth() = %#v, error = %v", depth, err)
	}
	klines, err := client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "1m", Limit: 1})
	if err != nil || len(klines) != 1 || klines[0].Close != "105" || klines[0].QuoteAssetVolume != "200" {
		t.Fatalf("FuturesKlines() = %#v, error = %v", klines, err)
	}
	ticker, err := client.FuturesTicker24hr(context.Background(), "BTCUSDT")
	if err != nil || ticker.PriceChange != "5" || ticker.PriceChangePercent != "5" || ticker.MarkPrice != "105.0" {
		t.Fatalf("FuturesTicker24hr() = %#v, error = %v", ticker, err)
	}
	price, err := client.FuturesTickerPrice(context.Background(), "BTCUSDT")
	if err != nil || price.Price != "105" || price.Time != 1700000000789 {
		t.Fatalf("FuturesTickerPrice() = %#v, error = %v", price, err)
	}
	bookTicker, err := client.FuturesBookTicker(context.Background(), "BTCUSDT")
	if err != nil || bookTicker.BidPrice != "104.9" || bookTicker.AskQty != "1.2" {
		t.Fatalf("FuturesBookTicker() = %#v, error = %v", bookTicker, err)
	}
	premium, err := client.FuturesPremiumIndex(context.Background(), "BTCUSDT")
	if err != nil || premium.MarkPrice != "105.0" || premium.IndexPrice != "104.8" || premium.LastFundingRate != "0.0001" {
		t.Fatalf("FuturesPremiumIndex() = %#v, error = %v", premium, err)
	}
}

func TestClientUSDTFuturesValidation(t *testing.T) {
	client := New(config.BitgetConfig{BaseURL: "https://example.com", HTTPTimeout: time.Second})
	if _, err := client.FuturesDepth(context.Background(), "BTCUSDT", 20); err == nil {
		t.Fatal("FuturesDepth() error = nil, want Bitget limit validation error")
	}
	if _, err := client.FuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSDT", Interval: "2h"}); err == nil {
		t.Fatal("FuturesKlines() error = nil, want interval validation error")
	}
	if _, err := client.FuturesTickerPrice(context.Background(), ""); err == nil {
		t.Fatal("FuturesTickerPrice() error = nil, want symbol validation error")
	}
}
