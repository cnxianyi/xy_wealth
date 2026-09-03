package binance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
)

func TestClientFuturesAccountEndpoints(t *testing.T) {
	const (
		apiKey    = "test-api-key"
		secretKey = "test-secret-key"
		timestamp = int64(1700000000000)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-MBX-APIKEY"); got != apiKey {
			t.Errorf("API key header = %q, want %q", got, apiKey)
		}
		query := r.URL.Query()
		signature := query.Get("signature")
		query.Del("signature")
		if got, want := signature, sign(query.Encode(), secretKey); got != want {
			t.Errorf("signature = %q, want %q", got, want)
		}
		if got := query.Get("timestamp"); got != "1700000000000" {
			t.Errorf("timestamp = %q, want 1700000000000", got)
		}
		if got := query.Get("recvWindow"); got != "5000" {
			t.Errorf("recvWindow = %q, want 5000", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/fapi/v3/balance":
			_, _ = w.Write([]byte(`[{"accountAlias":"um-account","asset":"USDT","balance":"100.00000000","crossWalletBalance":"90.00000000","crossUnPnl":"1.50000000","availableBalance":"88.50000000","maxWithdrawAmount":"88.50000000","marginAvailable":true,"updateTime":1700000001000},{"accountAlias":"um-account","asset":"USDC","balance":"0","crossWalletBalance":"0","crossUnPnl":"0","availableBalance":"0","maxWithdrawAmount":"0","marginAvailable":true,"updateTime":1700000001000}]`))
		case "/dapi/v1/balance":
			_, _ = w.Write([]byte(`[{"accountAlias":"cm-account","asset":"BTC","balance":"0.25000000","withdrawAvailable":"0.20000000","crossWalletBalance":"0.24000000","crossUnPnl":"0.01000000","availableBalance":"0.20000000","updateTime":1700000002000},{"accountAlias":"cm-account","asset":"ETH","balance":"0","withdrawAvailable":"0","crossWalletBalance":"0","crossUnPnl":"0","availableBalance":"0","updateTime":1700000002000}]`))
		case "/fapi/v3/positionRisk":
			if got := query.Get("symbol"); got != "" && got != "BTCUSDT" {
				t.Errorf("USDⓈ-M position symbol = %q, want empty or BTCUSDT", got)
			}
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","positionSide":"BOTH","positionAmt":"0.100","entryPrice":"60000.00","breakEvenPrice":"60001.00","markPrice":"60100.00","unRealizedProfit":"10.00","liquidationPrice":"50000.00","leverage":"20","marginType":"isolated","isolatedMargin":"300.00","isAutoAddMargin":"false","notional":"6010.00","isolatedWallet":"300.00","initialMargin":"300.50","maintMargin":"30.00","positionInitialMargin":"300.50","openOrderInitialMargin":"0","maxNotionalValue":"1000000","bidNotional":"0","askNotional":"0","adl":2,"updateTime":1700000003000}]`))
		case "/dapi/v1/positionRisk":
			if got := query.Get("pair"); got != "BTCUSD" {
				t.Errorf("COIN-M position pair = %q, want BTCUSD", got)
			}
			if got := query.Get("symbol"); got != "" {
				t.Errorf("COIN-M position symbol = %q, want empty", got)
			}
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSD_PERP","positionAmt":"2","entryPrice":"60000.00","breakEvenPrice":"60010.00","markPrice":"60100.00","unRealizedProfit":"0.001","liquidationPrice":"50000.00","leverage":"10","maxQty":"100","marginType":"cross","isolatedMargin":"0","isAutoAddMargin":false,"positionSide":"BOTH","updateTime":1700000004000,"notionalValue":"0.03327787"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BinanceConfig{
		FuturesBaseURL:      server.URL,
		CoinMFuturesBaseURL: server.URL,
		APIKey:              apiKey,
		SecretKey:           secretKey,
		RecvWindow:          5000,
		HTTPTimeout:         time.Second,
	})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	usdmBalances, err := client.USDSMFuturesAccountBalances(context.Background())
	if err != nil {
		t.Fatalf("USDSMFuturesAccountBalances() error = %v", err)
	}
	if len(usdmBalances) != 1 || usdmBalances[0].Asset != "USDT" || usdmBalances[0].AvailableBalance != "88.50000000" {
		t.Fatalf("USDⓈ-M balances = %#v", usdmBalances)
	}

	coinMBalances, err := client.COINMFuturesAccountBalances(context.Background())
	if err != nil {
		t.Fatalf("COINMFuturesAccountBalances() error = %v", err)
	}
	if len(coinMBalances) != 1 || coinMBalances[0].Asset != "BTC" || coinMBalances[0].WithdrawAvailable != "0.20000000" {
		t.Fatalf("COIN-M balances = %#v", coinMBalances)
	}

	usdmPositions, err := client.USDSMFuturesPositions(context.Background(), "")
	if err != nil {
		t.Fatalf("USDSMFuturesPositions(all) error = %v", err)
	}
	if len(usdmPositions) != 1 || usdmPositions[0].PositionAmount != "0.100" || usdmPositions[0].ADL != 2 || usdmPositions[0].IsAutoAddMargin == nil || *usdmPositions[0].IsAutoAddMargin {
		t.Fatalf("USDⓈ-M positions = %#v", usdmPositions)
	}

	coinMPositions, err := client.COINMFuturesPositions(context.Background(), "btcusd_perp")
	if err != nil {
		t.Fatalf("COINMFuturesPositions(symbol) error = %v", err)
	}
	if len(coinMPositions) != 1 || coinMPositions[0].Symbol != "BTCUSD_PERP" || coinMPositions[0].NotionalValue != "0.03327787" {
		t.Fatalf("COIN-M positions = %#v", coinMPositions)
	}
}

func TestClientFuturesAccountRequiresCredentials(t *testing.T) {
	client := New(config.BinanceConfig{FuturesBaseURL: "https://example.com", CoinMFuturesBaseURL: "https://example.com", HTTPTimeout: time.Second})
	checks := []struct {
		name string
		call func() error
	}{
		{name: "USDⓈ-M balances", call: func() error { _, err := client.USDSMFuturesAccountBalances(context.Background()); return err }},
		{name: "USDⓈ-M positions", call: func() error { _, err := client.USDSMFuturesPositions(context.Background(), "BTCUSDT"); return err }},
		{name: "COIN-M balances", call: func() error { _, err := client.COINMFuturesAccountBalances(context.Background()); return err }},
		{name: "COIN-M positions", call: func() error { _, err := client.COINMFuturesPositions(context.Background(), "BTCUSD_PERP"); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, ErrCredentialsMissing) {
				t.Fatalf("error = %v, want ErrCredentialsMissing", err)
			}
		})
	}
}
