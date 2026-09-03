package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDocsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	docs := NewDocs()
	router := gin.New()
	router.GET("/openapi.yaml", docs.OpenAPI)
	router.GET("/docs", docs.UI)

	tests := []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/openapi.yaml", contentType: "application/yaml", contains: "openapi: 3.1.0"},
		{path: "/docs", contentType: "text/html", contains: "Scalar.createApiReference"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, test.contentType) {
				t.Fatalf("Content-Type = %q, want prefix %q", got, test.contentType)
			}
			if !strings.Contains(response.Body.String(), test.contains) {
				t.Fatalf("response does not contain %q", test.contains)
			}
		})
	}
}
