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
		if excluded, hasExclude := c.GetQuery("exclude_zero"); hasExclude {
			value = excluded
			exists = true
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				writeSummaryBooleanError(c, "exclude_zero")
				return summary.Options{}, false
			}
			includeZero := !parsed
			return summary.Options{IncludeZero: &includeZero}, true
		}
	}
	if !exists {
		return summary.Options{}, true
	}
	includeZero, err := strconv.ParseBool(value)
	if err != nil {
		writeSummaryBooleanError(c, "include_zero")
		return summary.Options{}, false
	}
	return summary.Options{IncludeZero: &includeZero}, true
}

func writeSummaryBooleanError(c *gin.Context, parameter string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{
			"code":      "invalid_parameter",
			"message":   parameter + " must be a boolean",
			"parameter": parameter,
		},
	})
}
