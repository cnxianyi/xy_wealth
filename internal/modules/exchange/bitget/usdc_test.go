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

func TestClientUSDCMFuturesEndpoints(t *testing.T) {
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
			if got := r.URL.Query().Get("productType"); got != bitgetUSDCFuturesProductType {
				t.Errorf("contract productType = %q, want %s", got, bitgetUSDCFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCPERP","baseCoin":"BTC","quoteCoin":"USDC","supportMarginCoins":["USDC"],"volumePlace":"4","pricePlace":"1","symbolType":"perpetual","symbolStatus":"normal","maxLever":"100","minTradeNum":"0.0001","minTradeUSDT":"5","deliveryTime":"","launchTime":""}]}`))
		case "/api/v2/mix/market/merge-depth":
			if got := r.URL.Query().Get("productType"); got != bitgetUSDCFuturesProductType {
				t.Errorf("depth productType = %q, want %s", got, bitgetUSDCFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"asks":[[101.2,2.5]],"bids":[[100.8,3]],"ts":"1700000000456"}}`))
		case "/api/v2/mix/market/candles":
			if got := r.URL.Query().Get("productType"); got != bitgetUSDCFuturesProductType || r.URL.Query().Get("granularity") != "1m" {
				t.Errorf("candles query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[["1700000000000","100","110","90","105","2","200"]]}`))
		case "/api/v2/mix/market/ticker":
			if got := r.URL.Query().Get("productType"); got != bitgetUSDCFuturesProductType {
				t.Errorf("ticker productType = %q, want %s", got, bitgetUSDCFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCPERP","lastPr":"105","askPr":"105.1","bidPr":"104.9","bidSz":"1.5","askSz":"1.2","high24h":"110","low24h":"90","ts":"1700000000789","change24h":"0.05","baseVolume":"2","quoteVolume":"200","open24h":"100","indexPrice":"104.8","fundingRate":"0.0001","markPrice":"105.0"}]}`))
		case "/api/v2/mix/market/symbol-price":
			if got := r.URL.Query().Get("productType"); got != bitgetUSDCFuturesProductType {
				t.Errorf("price productType = %q, want %s", got, bitgetUSDCFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"BTCPERP","price":"105","indexPrice":"104.8","markPrice":"105.0","ts":"1700000000789"}]}`))
		case "/api/v2/mix/account/accounts":
			if got := r.URL.Query().Get("productType"); got != bitgetUSDCFuturesProductType {
				t.Errorf("account productType = %q, want %s", got, bitgetUSDCFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDC","locked":"0.01","available":"100","maxTransferOut":"99","accountEquity":"100.11","unrealizedPL":"0.01"}]}`))
		case "/api/v2/mix/position/all-position":
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDC","symbol":"BTCPERP","holdSide":"long","marginSize":"10","total":"1","leverage":"10","openPriceAvg":"60000","marginMode":"crossed","unrealizedPL":"0.1","liquidationPrice":"50000","markPrice":"61000","breakEvenPrice":"60010","uTime":"1700000001000","autoMargin":"off"}]}`))
		case "/api/v2/mix/position/single-position":
			if got := r.URL.Query().Get("symbol"); got != "BTCPERP" || r.URL.Query().Get("marginCoin") != "USDC" {
				t.Errorf("single position query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDC","symbol":"BTCPERP","holdSide":"short","marginSize":"20","total":"2","leverage":"5","openPriceAvg":"60000","marginMode":"isolated","unrealizedPL":"-0.1","liquidationPrice":"70000","markPrice":"59000","breakEvenPrice":"59990","uTime":"1700000002000","autoMargin":"on"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BitgetConfig{BaseURL: server.URL, APIKey: apiKey, SecretKey: secretKey, Passphrase: passphrase, HTTPTimeout: time.Second})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	info, err := client.USDCMFuturesExchangeInfo(context.Background())
	if err != nil || info.FuturesType != bitgetUSDCFuturesProductType || len(info.Symbols) != 1 || info.Symbols[0].MarginAsset != "USDC" {
		t.Fatalf("USDC-FUTURES exchange info = %#v, error = %v", info, err)
	}
	depth, err := client.USDCMFuturesDepth(context.Background(), "btcperp", 5)
	if err != nil || depth.Bids[0][0] != "100.8" || depth.TransactionTime != 1700000000456 {
		t.Fatalf("USDC-FUTURES depth = %#v, error = %v", depth, err)
	}
	klines, err := client.USDCMFuturesKlines(context.Background(), exchange.KlinesRequest{Symbol: "BTCPERP", Interval: "1m", Limit: 1})
	if err != nil || len(klines) != 1 || klines[0].Close != "105" {
		t.Fatalf("USDC-FUTURES klines = %#v, error = %v", klines, err)
	}
	ticker, err := client.USDCMFuturesTicker24hr(context.Background(), "BTCPERP")
	if err != nil || ticker.PriceChange != "5" || ticker.PriceChangePercent != "5" {
		t.Fatalf("USDC-FUTURES ticker = %#v, error = %v", ticker, err)
	}
	price, err := client.USDCMFuturesTickerPrice(context.Background(), "BTCPERP")
	if err != nil || price.Price != "105" {
		t.Fatalf("USDC-FUTURES price = %#v, error = %v", price, err)
	}
	book, err := client.USDCMFuturesBookTicker(context.Background(), "BTCPERP")
	if err != nil || book.BidPrice != "104.9" || book.AskQty != "1.2" {
		t.Fatalf("USDC-FUTURES book ticker = %#v, error = %v", book, err)
	}
	premium, err := client.USDCMFuturesPremiumIndex(context.Background(), "BTCPERP")
	if err != nil || premium.MarkPrice != "105.0" || premium.LastFundingRate != "0.0001" {
		t.Fatalf("USDC-FUTURES premium = %#v, error = %v", premium, err)
	}
	balances, err := client.USDCMFuturesAccountBalances(context.Background())
	if err != nil || len(balances) != 1 || balances[0].Asset != "USDC" || balances[0].Balance != "100.11" {
		t.Fatalf("USDC-FUTURES balances = %#v, error = %v", balances, err)
	}
	positions, err := client.USDCMFuturesPositions(context.Background(), "")
	if err != nil || len(positions) != 1 || positions[0].PositionSide != "LONG" {
		t.Fatalf("USDC-FUTURES all positions = %#v, error = %v", positions, err)
	}
	positions, err = client.USDCMFuturesPositions(context.Background(), "btcperp")
	if err != nil || len(positions) != 1 || positions[0].Symbol != "BTCPERP" || positions[0].PositionSide != "SHORT" || positions[0].IsAutoAddMargin == nil || !*positions[0].IsAutoAddMargin {
		t.Fatalf("USDC-FUTURES single position = %#v, error = %v", positions, err)
	}
}

func TestClientUSDCMFuturesAccountRequiresCredentials(t *testing.T) {
	client := New(config.BitgetConfig{BaseURL: "https://example.com", HTTPTimeout: time.Second})
	if _, err := client.USDCMFuturesAccountBalances(context.Background()); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("account balances error = %v, want ErrCredentialsMissing", err)
	}
	if _, err := client.USDCMFuturesPositions(context.Background(), "BTCPERP"); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("account positions error = %v, want ErrCredentialsMissing", err)
	}
}
