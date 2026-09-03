package bitget

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
)

func TestClientUSDTFuturesAccountEndpoints(t *testing.T) {
	const (
		apiKey     = "bitget-api-key"
		secretKey  = "bitget-secret-key"
		passphrase = "bitget-passphrase"
		timestamp  = int64(1700000000000)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(accessKeyHeader); got != apiKey {
			t.Errorf("API key header = %q, want %q", got, apiKey)
		}
		if got := r.Header.Get(accessPassphraseHeader); got != passphrase {
			t.Errorf("passphrase header = %q, want %q", got, passphrase)
		}
		payload := r.Header.Get(accessTimestampHeader) + http.MethodGet + r.URL.Path + "?" + r.URL.RawQuery
		if got, want := r.Header.Get(accessSignHeader), sign(payload, secretKey); got != want {
			t.Errorf("signature = %q, want %q", got, want)
		}
		if got := r.Header.Get(accessTimestampHeader); got != "1700000000000" {
			t.Errorf("timestamp header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/mix/account/accounts":
			if got := r.URL.Query().Get("productType"); got != bitgetUSDTFuturesProductType {
				t.Errorf("account productType = %q, want %s", got, bitgetUSDTFuturesProductType)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDT","locked":"1","available":"99","crossedMaxAvailable":"100","maxTransferOut":"98","accountEquity":"101","unrealizedPL":"2"},{"marginCoin":"USDT","locked":"0","available":"0","crossedMaxAvailable":"0","maxTransferOut":"0","accountEquity":"0","unrealizedPL":"0"}]}`))
		case "/api/v2/mix/position/all-position":
			if got := r.URL.Query().Get("symbol"); got != "" || r.URL.Query().Get("marginCoin") != "" {
				t.Errorf("all-position filters = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDT","symbol":"BTCUSDT","holdSide":"short","marginSize":"10","available":"0.1","locked":"0","total":"0.1","leverage":"10","openPriceAvg":"60000","marginMode":"crossed","unrealizedPL":"-2","liquidationPrice":"70000","markPrice":"61000","breakEvenPrice":"60010","uTime":"1700000001000","autoMargin":"off"}]}`))
		case "/api/v2/mix/position/single-position":
			if got := r.URL.Query().Get("symbol"); got != "ETHUSDT" || r.URL.Query().Get("marginCoin") != "USDT" {
				t.Errorf("single-position filters = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDT","symbol":"ETHUSDT","holdSide":"long","marginSize":"5","available":"1","locked":"0","total":"1","leverage":"5","openPriceAvg":"3000","marginMode":"isolated","unrealizedPL":"3","liquidationPrice":"2000","markPrice":"3100","breakEvenPrice":"3010","uTime":"1700000002000","autoMargin":"on"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(configForBitgetAccount(server.URL, apiKey, secretKey, passphrase))
	client.now = func() time.Time { return time.UnixMilli(timestamp) }

	balances, err := client.USDSMFuturesAccountBalances(context.Background())
	if err != nil || len(balances) != 1 || balances[0].Asset != "USDT" || balances[0].Balance != "101" || balances[0].AvailableBalance != "99" {
		t.Fatalf("USDT-FUTURES balances = %#v, error = %v", balances, err)
	}
	positions, err := client.USDSMFuturesPositions(context.Background(), "")
	if err != nil || len(positions) != 1 || positions[0].PositionSide != "SHORT" || positions[0].MarginType != "cross" || positions[0].PositionAmount != "0.1" || positions[0].IsAutoAddMargin == nil || *positions[0].IsAutoAddMargin {
		t.Fatalf("all positions = %#v, error = %v", positions, err)
	}
	positions, err = client.USDSMFuturesPositions(context.Background(), "ethusdt")
	if err != nil || len(positions) != 1 || positions[0].Symbol != "ETHUSDT" || positions[0].PositionSide != "LONG" || positions[0].MarginType != "isolated" || positions[0].IsAutoAddMargin == nil || !*positions[0].IsAutoAddMargin {
		t.Fatalf("single position = %#v, error = %v", positions, err)
	}
}

func TestClientUSDTFuturesAccountRequiresCredentials(t *testing.T) {
	client := New(configForBitgetAccount("https://example.com", "", "", ""))
	if _, err := client.USDSMFuturesAccountBalances(context.Background()); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("account balances error = %v, want ErrCredentialsMissing", err)
	}
	if _, err := client.USDSMFuturesPositions(context.Background(), "BTCUSDT"); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("account positions error = %v, want ErrCredentialsMissing", err)
	}
}

func configForBitgetAccount(baseURL, apiKey, secretKey, passphrase string) config.BitgetConfig {
	return config.BitgetConfig{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		SecretKey:   secretKey,
		Passphrase:  passphrase,
		HTTPTimeout: time.Second,
	}
}
