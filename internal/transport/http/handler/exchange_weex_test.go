package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/weex/spot/ping", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("Weex Spot status = %d, want 200: %s", response.Code, response.Body.String())
	}
}

type stubWeexSpotProvider struct{ stubSpotProvider }

func (stubWeexSpotProvider) Name() string { return "weex" }
