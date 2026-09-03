package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/cnxianyi/xy_wealth/internal/modules/summary"
	"github.com/gin-gonic/gin"
)

func TestSummaryIncludeZeroQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := summary.New([]exchange.Provider{summaryQueryExchange{}}, nil)
	handler := NewSummary(service)
	router := gin.New()
	router.GET("/api/v1/summary", handler.All)

	tests := []struct {
		name       string
		path       string
		statusCode int
		contains   string
		notContain string
	}{
		{name: "default preserves provider response", path: "/api/v1/summary", statusCode: http.StatusOK, contains: "ZERO"},
		{name: "false filters zero values", path: "/api/v1/summary?include_zero=false", statusCode: http.StatusOK, contains: "BTC", notContain: "ZERO"},
		{name: "true keeps zero values", path: "/api/v1/summary?include_zero=true", statusCode: http.StatusOK, contains: "ZERO"},
		{name: "invalid boolean", path: "/api/v1/summary?include_zero=maybe", statusCode: http.StatusBadRequest, contains: "invalid_parameter"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			body := response.Body.String()
			if response.Code != test.statusCode || !strings.Contains(body, test.contains) || (test.notContain != "" && strings.Contains(body, test.notContain)) {
				t.Fatalf("response = %d: %s", response.Code, body)
			}
		})
	}
}

type summaryQueryExchange struct{}

func (summaryQueryExchange) Name() string { return "query-test" }

func (summaryQueryExchange) Balances(context.Context) ([]asset.Balance, error) {
	return []asset.Balance{
		{Symbol: "ZERO", Free: "0", Locked: "0", Total: "0"},
		{Symbol: "BTC", Free: "1", Locked: "0", Total: "1"},
	}, nil
}
