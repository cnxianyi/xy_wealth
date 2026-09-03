package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestExchangeBinanceFuturesAccountHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := stubBinanceAccountProvider{}
	handler := NewExchange([]exchange.Provider{provider}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/futures/usdm/account/balances", handler.USDSMFuturesAccountBalances)
	router.GET("/exchanges/:provider/futures/usdm/account/positions", handler.USDSMFuturesAccountPositions)
	router.GET("/exchanges/:provider/futures/coinm/account/balances", handler.COINMFuturesAccountBalances)
	router.GET("/exchanges/:provider/futures/coinm/account/positions", handler.COINMFuturesAccountPositions)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "USDⓈ-M balances", path: "/exchanges/binance/futures/usdm/account/balances", want: `"asset":"USDT"`},
		{name: "USDⓈ-M positions", path: "/exchanges/binance/futures/usdm/account/positions?symbol=BTCUSDT", want: `"symbol":"BTCUSDT"`},
		{name: "COIN-M balances", path: "/exchanges/binance/futures/coinm/account/balances", want: `"asset":"BTC"`},
		{name: "COIN-M positions", path: "/exchanges/binance/futures/coinm/account/positions?symbol=BTCUSD_PERP", want: `"symbol":"BTCUSD_PERP"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), test.want) {
				t.Fatalf("response = %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestExchangeBinanceFuturesAccountCapabilityErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewExchange([]exchange.Provider{stubSpotProvider{}}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/futures/usdm/account/balances", handler.USDSMFuturesAccountBalances)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/missing/futures/usdm/account/balances", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing provider status = %d, want 404: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/binance/futures/usdm/account/balances", nil))
	if response.Code != http.StatusNotImplemented || !strings.Contains(response.Body.String(), `"code":"endpoint_not_supported"`) {
		t.Fatalf("unsupported capability response = %d: %s", response.Code, response.Body.String())
	}
}

type stubBinanceAccountProvider struct{ stubSpotProvider }

func (stubBinanceAccountProvider) Name() string { return "binance" }

func (stubBinanceAccountProvider) USDSMFuturesAccountBalances(context.Context) ([]exchange.FuturesAccountBalance, error) {
	return []exchange.FuturesAccountBalance{{Asset: "USDT", Balance: "100"}}, nil
}

func (stubBinanceAccountProvider) USDSMFuturesPositions(context.Context, string) ([]exchange.FuturesPosition, error) {
	return []exchange.FuturesPosition{{Symbol: "BTCUSDT", PositionSide: "BOTH"}}, nil
}

func (stubBinanceAccountProvider) COINMFuturesAccountBalances(context.Context) ([]exchange.FuturesAccountBalance, error) {
	return []exchange.FuturesAccountBalance{{Asset: "BTC", Balance: "1"}}, nil
}

func (stubBinanceAccountProvider) COINMFuturesPositions(context.Context, string) ([]exchange.FuturesPosition, error) {
	return []exchange.FuturesPosition{{Symbol: "BTCUSD_PERP", PositionSide: "BOTH"}}, nil
}
