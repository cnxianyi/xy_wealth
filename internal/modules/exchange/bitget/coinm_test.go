package bitget

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

func TestClientCOINFuturesEndpoints(t *testing.T) {
	const (
		apiKey     = "bitget-api-key"
		secretKey  = "bitget-secret-key"
		passphrase = "bitget-passphrase"
		timestamp  = int64(1700000000000)
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/mix/account/accounts" || r.URL.Path == "/api/v2/mix/position/all-position" || r.URL.Path == "/api/v2/mix/position/single-position" {
			if got := r.Header.Get(accessKeyHeader); got != apiKey {
				t.Errorf("API key header = %q, want %q", got, apiKey)
			}
			payload := r.Header.Get(accessTimestampHeader) + http.MethodGet + r.URL.Path + "?" + r.URL.RawQuery
			if got, want := r.Header.Get(accessSignHeader), sign(payload, secretKey); got != want {
				t.Errorf("signature = %q, want %q", got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/public/time":
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"serverTime":"1700000000123"}}`))
		case "/api/v2/mix/market/contracts":
			if got := r.URL.Query().Get("productType"); got != bitgetCOINFuturesProductType {
				t.Errorf("contract productType = %q, want %s", got, bitgetCOINFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSD","baseCoin":"BTC","quoteCoin":"USD","supportMarginCoins":["BTC"],"volumePlace":"4","pricePlace":"1","symbolType":"perpetual","symbolStatus":"normal","maxLever":"125","minTradeNum":"1","minTradeUSDT":"5","deliveryTime":"","launchTime":""}]}`))
		case "/api/v2/mix/market/merge-depth":
			if got := r.URL.Query().Get("productType"); got != bitgetCOINFuturesProductType {
				t.Errorf("depth productType = %q, want %s", got, bitgetCOINFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"asks":[[101.2,2.5]],"bids":[[100.8,3]],"ts":"1700000000456"}}`))
		case "/api/v2/mix/market/candles":
			if got := r.URL.Query().Get("productType"); got != bitgetCOINFuturesProductType || r.URL.Query().Get("granularity") != "1m" {
				t.Errorf("candles query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[["1700000000000","100","110","90","105","2","200"]]}`))
		case "/api/v2/mix/market/ticker":
			if got := r.URL.Query().Get("productType"); got != bitgetCOINFuturesProductType {
				t.Errorf("ticker productType = %q, want %s", got, bitgetCOINFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSD","lastPr":"105","askPr":"105.1","bidPr":"104.9","bidSz":"1.5","askSz":"1.2","high24h":"110","low24h":"90","ts":"1700000000789","change24h":"0.05","baseVolume":"2","quoteVolume":"200","open24h":"100","indexPrice":"104.8","fundingRate":"0.0001","markPrice":"105.0"}]}`))
		case "/api/v2/mix/market/symbol-price":
			if got := r.URL.Query().Get("productType"); got != bitgetCOINFuturesProductType {
				t.Errorf("price productType = %q, want %s", got, bitgetCOINFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCUSD","price":"105","indexPrice":"104.8","markPrice":"105.0","ts":"1700000000789"}]}`))
		case "/api/v2/mix/account/accounts":
			if got := r.URL.Query().Get("productType"); got != bitgetCOINFuturesProductType {
				t.Errorf("account productType = %q, want %s", got, bitgetCOINFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"BTC","locked":"0.01","available":"0.1","maxTransferOut":"0.09","accountEquity":"0.11","unrealizedPL":"0.01"}]}`))
		case "/api/v2/mix/position/all-position":
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"BTC","symbol":"BTCUSD","holdSide":"long","marginSize":"0.01","total":"1","leverage":"10","openPriceAvg":"60000","marginMode":"crossed","unrealizedPL":"0.1","liquidationPrice":"50000","markPrice":"61000","breakEvenPrice":"60010","uTime":"1700000001000","autoMargin":"off"}]}`))
		case "/api/v2/mix/position/single-position":
			if got := r.URL.Query().Get("symbol"); got != "BTCUSD_PERP" || r.URL.Query().Get("marginCoin") != "BTC" {
				t.Errorf("single position query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"BTC","symbol":"BTCUSD_PERP","holdSide":"short","marginSize":"0.02","total":"2","leverage":"5","openPriceAvg":"60000","marginMode":"isolated","unrealizedPL":"-0.1","liquidationPrice":"70000","markPrice":"59000","breakEvenPrice":"59990","uTime":"1700000002000","autoMargin":"on"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BitgetConfig{BaseURL: server.URL, APIKey: apiKey, SecretKey: secretKey, Passphrase: passphrase, HTTPTimeout: time.Second})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	info, err := client.CoinMFuturesExchangeInfo(context.Background())
	if err != nil || len(info.Symbols) != 1 || info.Symbols[0].MarginAsset != "BTC" || info.Symbols[0].ContractType != "PERPETUAL" {
		t.Fatalf("COIN-FUTURES exchange info = %#v, error = %v", info, err)
	}
	depth, err := client.CoinMFuturesDepth(context.Background(), "btcusd", 5)
	if err != nil || depth.Bids[0][0] != "100.8" || depth.TransactionTime != 1700000000456 {
		t.Fatalf("COIN-FUTURES depth = %#v, error = %v", depth, err)
	}
	klines, err := client.CoinMFuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCUSD", Interval: "1m", Limit: 1})
	if err != nil || len(klines) != 1 || klines[0].Close != "105" {
		t.Fatalf("COIN-FUTURES klines = %#v, error = %v", klines, err)
	}
	ticker, err := client.CoinMFuturesTicker24hr(context.Background(), "BTCUSD")
	if err != nil || ticker.PriceChange != "5" || ticker.PriceChangePercent != "5" || ticker.WeightedAvgPrice != "100" {
		t.Fatalf("COIN-FUTURES ticker = %#v, error = %v", ticker, err)
	}
	price, err := client.CoinMFuturesTickerPrice(context.Background(), "BTCUSD")
	if err != nil || price.Pair != "BTCUSD" || price.Price != "105" {
		t.Fatalf("COIN-FUTURES price = %#v, error = %v", price, err)
	}
	book, err := client.CoinMFuturesBookTicker(context.Background(), "BTCUSD")
	if err != nil || book.BidPrice != "104.9" || book.AskQty != "1.2" {
		t.Fatalf("COIN-FUTURES book ticker = %#v, error = %v", book, err)
	}
	premium, err := client.CoinMFuturesPremiumIndex(context.Background(), "BTCUSD")
	if err != nil || premium.MarkPrice != "105.0" || premium.LastFundingRate != "0.0001" {
		t.Fatalf("COIN-FUTURES premium = %#v, error = %v", premium, err)
	}
	balances, err := client.COINMFuturesAccountBalances(context.Background())
	if err != nil || len(balances) != 1 || balances[0].Asset != "BTC" || balances[0].Balance != "0.11" {
		t.Fatalf("COIN-FUTURES balances = %#v, error = %v", balances, err)
	}
	positions, err := client.COINMFuturesPositions(context.Background(), "")
	if err != nil || len(positions) != 1 || positions[0].PositionSide != "LONG" {
		t.Fatalf("COIN-FUTURES all positions = %#v, error = %v", positions, err)
	}
	positions, err = client.COINMFuturesPositions(context.Background(), "btcusd_perp")
	if err != nil || len(positions) != 1 || positions[0].Symbol != "BTCUSD_PERP" || positions[0].PositionSide != "SHORT" || positions[0].IsAutoAddMargin == nil || !*positions[0].IsAutoAddMargin {
		t.Fatalf("COIN-FUTURES single position = %#v, error = %v", positions, err)
	}
}

func TestClientCOINFuturesAccountRequiresCredentials(t *testing.T) {
	client := New(config.BitgetConfig{BaseURL: "https://example.com", HTTPTimeout: time.Second})
	if _, err := client.COINMFuturesAccountBalances(context.Background()); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("account balances error = %v, want ErrCredentialsMissing", err)
	}
	if _, err := client.COINMFuturesPositions(context.Background(), "BTCUSD_PERP"); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("account positions error = %v, want ErrCredentialsMissing", err)
	}
}
