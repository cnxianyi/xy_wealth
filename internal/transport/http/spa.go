package httptransport

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var reservedWebPrefixes = []string{"/api/", "/health/", "/openapi.yaml", "/docs"}

func registerSPA(router *gin.Engine, staticDir string, log *zap.Logger) {
	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Warn("inspect web static directory failed", zap.String("directory", staticDir), zap.Error(err))
		} else {
			log.Info("web frontend is not built; static routes disabled", zap.String("directory", staticDir))
		}
		return
	}

	serveIndex := func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File(indexPath)
	}
	router.GET("/", serveIndex)
	router.StaticFS("/assets", http.Dir(filepath.Join(staticDir, "assets")))
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet || isReservedWebPath(c.Request.URL.Path) || !acceptsHTML(c.Request) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "route not found"}})
			return
		}
		requested := strings.TrimPrefix(filepath.Clean(c.Request.URL.Path), string(filepath.Separator))
		if requested != "." && requested != "" {
			candidate := filepath.Join(staticDir, requested)
			if relative, err := filepath.Rel(staticDir, candidate); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					c.File(candidate)
					return
				}
			}
		}
		serveIndex(c)
	})
}

func isReservedWebPath(path string) bool {
	for _, prefix := range reservedWebPrefixes {
		if path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func acceptsHTML(request *http.Request) bool {
	accept := request.Header.Get("Accept")
	return accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}
