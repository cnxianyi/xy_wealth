package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestExchangeRoutesWeexSpotCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := stubWeexSpotProvider{}
	handler := NewExchange([]exchange.Provider{provider}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/spot/ping", handler.Ping)
	router.GET("/exchanges/:provider/futures/usdm/ping", handler.FuturesPing)
	router.GET("/exchanges/:provider/futures/usdm/positions", handler.ContractPositions)
	router.GET("/exchanges/:provider/futures/usdm/balances", handler.ContractBalances)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/weex/spot/ping", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("Weex Spot status = %d, want 200: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/weex/futures/usdm/ping", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("Weex Contract status = %d, want 200: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/weex/futures/usdm/positions?symbol=BTCUSDT", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"symbol":"BTCUSDT"`) {
		t.Fatalf("Weex Contract positions response = %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/weex/futures/usdm/balances", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"total":"100.5"`) {
		t.Fatalf("Weex Contract balances response = %d: %s", response.Code, response.Body.String())
	}
}

type stubWeexSpotProvider struct{ stubSpotProvider }

func (stubWeexSpotProvider) Name() string { return "weex" }

func (stubWeexSpotProvider) ContractPositions(context.Context, string) ([]exchange.ContractPosition, error) {
	return []exchange.ContractPosition{{Symbol: "BTCUSDT", Side: "LONG"}}, nil
}

func (stubWeexSpotProvider) ContractBalances(context.Context) ([]asset.Balance, error) {
	return []asset.Balance{{Symbol: "USDT", Free: "90.5", Locked: "10", Total: "100.5"}}, nil
}
