package httptransport

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRegisterSPAServesIndexAssetsAndFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("<h1>xy-wealth</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "assets", "app.js"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	registerSPA(router, directory, zap.NewNop())
	for _, test := range []struct {
		path, accept, contains string
		status                 int
	}{
		{path: "/", accept: "text/html", status: http.StatusOK, contains: "xy-wealth"},
		{path: "/holdings", accept: "text/html", status: http.StatusOK, contains: "xy-wealth"},
		{path: "/assets/app.js", accept: "*/*", status: http.StatusOK, contains: "app"},
		{path: "/api/v1/missing", accept: "text/html", status: http.StatusNotFound, contains: "not_found"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request.Header.Set("Accept", test.accept)
		router.ServeHTTP(response, request)
		if response.Code != test.status || !strings.Contains(response.Body.String(), test.contains) {
			t.Errorf("GET %s = %d %q, want %d containing %q", test.path, response.Code, response.Body.String(), test.status, test.contains)
		}
	}
}

func TestRegisterSPADisabledWithoutBuild(t *testing.T) {
	router := gin.New()
	registerSPA(router, filepath.Join(t.TempDir(), "missing"), zap.NewNop())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("GET / without build = %d, want 404", response.Code)
	}
}
