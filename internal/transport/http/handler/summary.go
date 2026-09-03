package handler

import (
	"net/http"
	"strconv"

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
	options, ok := parseSummaryOptions(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, h.service.GetWithOptions(c.Request.Context(), options))
}

func (h *Summary) Exchanges(c *gin.Context) {
	options, ok := parseSummaryOptions(c)
	if !ok {
		return
	}
	data := h.service.GetWithOptions(c.Request.Context(), options)
	c.JSON(http.StatusOK, gin.H{"generated_at": data.GeneratedAt, "exchanges": data.Exchanges})
}

func (h *Summary) Banks(c *gin.Context) {
	options, ok := parseSummaryOptions(c)
	if !ok {
		return
	}
	data := h.service.GetWithOptions(c.Request.Context(), options)
	c.JSON(http.StatusOK, gin.H{"generated_at": data.GeneratedAt, "banks": data.Banks})
}

func parseSummaryOptions(c *gin.Context) (summary.Options, bool) {
	value, exists := c.GetQuery("include_zero")
	if !exists {
		return summary.Options{}, true
	}
	includeZero, err := strconv.ParseBool(value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":      "invalid_parameter",
				"message":   "include_zero must be a boolean",
				"parameter": "include_zero",
			},
		})
		return summary.Options{}, false
	}
	return summary.Options{IncludeZero: &includeZero}, true
}
