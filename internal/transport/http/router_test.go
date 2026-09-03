package httptransport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cnxianyi/xy_wealth/internal/config"
	authmodule "github.com/cnxianyi/xy_wealth/internal/modules/auth"
	"github.com/cnxianyi/xy_wealth/internal/modules/summary"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestRouterProtectsExistingEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	cfg := config.Config{App: config.AppConfig{Environment: "test"}, Auth: config.AuthConfig{Secret: "secret", TokenTTL: time.Hour}}
	authService := authmodule.New(cfg.Auth, redisClient)
	router := NewRouter(cfg, zap.NewNop(), nil, redisClient, nil, summary.New(nil, nil), authService)

	public := map[string]bool{
		http.MethodGet + " /docs":               true,
		http.MethodGet + " /health/live":        true,
		http.MethodPost + " /api/v1/auth/login": true,
	}
	protectedCount := 0
	for _, route := range router.Routes() {
		if public[route.Method+" "+route.Path] {
			continue
		}
		protectedCount++
		path := strings.ReplaceAll(route.Path, ":provider", "binance")
		request := httptest.NewRequest(route.Method, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without x-token = %d, want 401", route.Method, route.Path, response.Code)
		}
	}
	if protectedCount < 50 {
		t.Fatalf("protected route count = %d, want at least 50", protectedCount)
	}
}
