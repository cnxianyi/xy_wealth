package handler

import (
	"net/http"

	"github.com/cnxianyi/xy_wealth/api"
	"github.com/gin-gonic/gin"
)

type Docs struct{}

func NewDocs() *Docs { return &Docs{} }

func (h *Docs) OpenAPI(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", api.Specification())
}

func (h *Docs) UI(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, "text/html; charset=utf-8", api.Documentation())
}
