package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cnxianyi/xy_wealth/internal/config"
	authmodule "github.com/cnxianyi/xy_wealth/internal/modules/auth"
	"github.com/cnxianyi/xy_wealth/internal/transport/http/handler"
	appmiddleware "github.com/cnxianyi/xy_wealth/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestAuthHTTPLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	service := authmodule.New(config.AuthConfig{Secret: "correct-secret", TokenTTL: time.Hour}, redisClient)
	log := zap.NewNop()
	h := handler.NewAuth(service, log)
	router := gin.New()
	router.POST("/api/v1/auth/login", h.Login)
	protected := router.Group("/api/v1/auth", appmiddleware.XToken(service, log))
	protected.GET("/session", h.Session)
	protected.POST("/logout", h.Logout)

	response := performJSON(router, http.MethodPost, "/api/v1/auth/login", `{"secret":"wrong"}`, "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong login status = %d, want 401: %s", response.Code, response.Body.String())
	}
	response = performJSON(router, http.MethodPost, "/api/v1/auth/login", `{"secret":"correct-secret"}`, "")
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var session struct {
		Token string `json:"x_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil || session.Token == "" {
		t.Fatalf("decode login response = %v, body = %s", err, response.Body.String())
	}

	response = performJSON(router, http.MethodGet, "/api/v1/auth/session", "", "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("session without token status = %d, want 401", response.Code)
	}
	response = performJSON(router, http.MethodGet, "/api/v1/auth/session", "", session.Token)
	if response.Code != http.StatusOK {
		t.Fatalf("session status = %d, want 200: %s", response.Code, response.Body.String())
	}
	response = performJSON(router, http.MethodPost, "/api/v1/auth/logout", "", session.Token)
	if response.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want 200: %s", response.Code, response.Body.String())
	}
	response = performJSON(router, http.MethodGet, "/api/v1/auth/session", "", session.Token)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("session after logout status = %d, want 401", response.Code)
	}
}

func performJSON(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("x-token", token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
