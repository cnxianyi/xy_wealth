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

func TestExchangeBitgetSpotRoutesUseIndependentProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewExchange([]exchange.Provider{stubBitgetSpotProvider{}}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/spot/ping", handler.Ping)
	router.GET("/exchanges/:provider/futures/usdm/ping", handler.FuturesPing)
	router.GET("/exchanges/:provider/futures/usdm/positions", handler.ContractPositions)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/bitget/spot/ping", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"provider":"bitget"`) {
		t.Fatalf("response = %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/bitget/futures/usdm/ping", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"provider":"bitget"`) {
		t.Fatalf("futures response = %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/bitget/futures/usdm/positions", nil))
	if response.Code != http.StatusNotImplemented || !strings.Contains(response.Body.String(), `"code":"endpoint_not_supported"`) {
		t.Fatalf("unsupported futures account response = %d: %s", response.Code, response.Body.String())
	}
}

func TestExchangeBitgetUSDTAccountRoutesUseAccountCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewExchange([]exchange.Provider{stubBitgetAccountProvider{}}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/futures/usdm/account/balances", handler.USDSMFuturesAccountBalances)
	router.GET("/exchanges/:provider/futures/usdm/account/positions", handler.USDSMFuturesAccountPositions)

	for _, path := range []string{
		"/exchanges/bitget/futures/usdm/account/balances",
		"/exchanges/bitget/futures/usdm/account/positions?symbol=BTCUSDT",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"provider":"bitget"`) {
			t.Fatalf("path %s response = %d: %s", path, response.Code, response.Body.String())
		}
	}
}

type stubBitgetSpotProvider struct{ stubSpotProvider }

func (stubBitgetSpotProvider) Name() string { return "bitget" }

type stubBitgetAccountProvider struct{ stubBitgetSpotProvider }

func (stubBitgetAccountProvider) USDSMFuturesAccountBalances(context.Context) ([]exchange.FuturesAccountBalance, error) {
	return []exchange.FuturesAccountBalance{{Asset: "USDT", Balance: "100"}}, nil
}

func (stubBitgetAccountProvider) USDSMFuturesPositions(context.Context, string) ([]exchange.FuturesPosition, error) {
	return []exchange.FuturesPosition{{Symbol: "BTCUSDT", PositionSide: "LONG"}}, nil
}
