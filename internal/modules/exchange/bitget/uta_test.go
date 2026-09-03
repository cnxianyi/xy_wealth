package bitget

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
)

func TestClientUnifiedAccountFallback(t *testing.T) {
	const (
		apiKey     = "bitget-api-key"
		secretKey  = "bitget-secret-key"
		passphrase = "bitget-passphrase"
		timestamp  = int64(1700000000000)
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/spot/account/assets" || r.URL.Path == "/api/v2/mix/account/accounts" || r.URL.Path == "/api/v2/mix/position/single-position" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"40085","msg":"You are in Unified Account mode"}`))
			return
		}
		if r.URL.Path == "/api/v3/account/assets" || r.URL.Path == "/api/v3/position/current-position" {
			if got := r.Header.Get(accessKeyHeader); got != apiKey {
				t.Errorf("API key header = %q, want %q", got, apiKey)
			}
			if got := r.Header.Get(accessPassphraseHeader); got != passphrase {
				t.Errorf("passphrase header = %q, want %q", got, passphrase)
			}
			payload := r.Header.Get(accessTimestampHeader) + http.MethodGet + r.URL.Path
			if r.URL.RawQuery != "" {
				payload += "?" + r.URL.RawQuery
			}
			if got, want := r.Header.Get(accessSignHeader), sign(payload, secretKey); got != want {
				t.Errorf("signature = %q, want %q", got, want)
			}
		}
		switch r.URL.Path {
		case "/api/v3/account/assets":
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"accountEquity":"111","assets":[{"coin":"USDT","equity":"101","balance":"100","available":"99","locked":"1"},{"coin":"USDC","equity":"20","balance":"20","available":"20","locked":"0"}]}}`))
		case "/api/v3/position/current-position":
			category := r.URL.Query().Get("category")
			symbol := map[string]string{"USDT-FUTURES": "BTCUSDT", "USDC-FUTURES": "BTCPERP", "COIN-FUTURES": "BTCUSD"}[category]
			marginCoin := map[string]string{"USDT-FUTURES": "USDT", "USDC-FUTURES": "USDC", "COIN-FUTURES": "BTC"}[category]
			if got := r.URL.Query().Get("symbol"); got != symbol {
				t.Errorf("UTA position symbol = %q, want %q", got, symbol)
			}
			_, _ = fmt.Fprintf(w, `{"code":"00000","msg":"success","data":{"list":[{"category":"%s","symbol":"%s","marginCoin":"%s","posSide":"long","positionBalance":"10","available":"1","frozen":"0","total":"1","leverage":"10","avgPrice":"60000","marginMode":"crossed","unrealisedPnl":"1","liquidationPrice":"50000","markPrice":"61000","breakEvenPrice":"60010","updatedTime":"1700000001000"}]}}`, category, symbol, marginCoin)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(config.BitgetConfig{BaseURL: server.URL, APIKey: apiKey, SecretKey: secretKey, Passphrase: passphrase, HTTPTimeout: time.Second})
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	spot, err := client.Balances(context.Background())
	if err != nil || len(spot) != 2 || spot[0].Symbol != "USDT" || spot[0].Free != "99" || spot[0].Locked != "1" || spot[0].Total != "100" {
		t.Fatalf("UTA spot balances = %#v, error = %v", spot, err)
	}
	for _, check := range []struct {
		name string
		call func() ([]string, error)
	}{
		{name: "USDT", call: func() ([]string, error) {
			balances, err := client.USDSMFuturesAccountBalances(context.Background())
			if err != nil {
				return nil, err
			}
			return []string{balances[0].Asset, balances[0].Balance}, nil
		}},
		{name: "USDC", call: func() ([]string, error) {
			balances, err := client.USDCMFuturesAccountBalances(context.Background())
			if err != nil {
				return nil, err
			}
			return []string{balances[1].Asset, balances[1].Balance}, nil
		}},
		{name: "COIN", call: func() ([]string, error) {
			balances, err := client.COINMFuturesAccountBalances(context.Background())
			if err != nil {
				return nil, err
			}
			return []string{balances[0].Asset, balances[0].Balance}, nil
		}},
	} {
		t.Run(check.name, func(t *testing.T) {
			values, err := check.call()
			if err != nil || len(values) != 2 || values[1] == "" {
				t.Fatalf("UTA %s balances = %#v, error = %v", check.name, values, err)
			}
		})
	}

	positions, err := client.USDSMFuturesPositions(context.Background(), "BTCUSDT")
	if err != nil || len(positions) != 1 || positions[0].PositionSide != "LONG" || positions[0].MarginType != "cross" || positions[0].UpdateTime != 1700000001000 {
		t.Fatalf("UTA USDT positions = %#v, error = %v", positions, err)
	}
	positions, err = client.USDCMFuturesPositions(context.Background(), "BTCPERP")
	if err != nil || len(positions) != 1 || positions[0].Symbol != "BTCPERP" {
		t.Fatalf("UTA USDC positions = %#v, error = %v", positions, err)
	}
	positions, err = client.COINMFuturesPositions(context.Background(), "BTCUSD")
	if err != nil || len(positions) != 1 || positions[0].Symbol != "BTCUSD" {
		t.Fatalf("UTA COIN positions = %#v, error = %v", positions, err)
	}
}
