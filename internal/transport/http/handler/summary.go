package handler

import (
	"net/http"

	"github.com/cnxianyi/xy_wealth/internal/modules/summary"
	"github.com/gin-gonic/gin"
)

type Summary struct {
	service *summary.Service
}

func NewSummary(service *summary.Service) *Summary {
	return &Summary{service: service}
}

func (h *Summary) All(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.Get(c.Request.Context()))
}

func (h *Summary) Exchanges(c *gin.Context) {
	data := h.service.Get(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"generated_at": data.GeneratedAt, "exchanges": data.Exchanges})
}

func (h *Summary) Banks(c *gin.Context) {
	data := h.service.Get(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"generated_at": data.GeneratedAt, "banks": data.Banks})
}
