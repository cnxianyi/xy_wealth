package handler

import (
	"net/http"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Exchange struct {
	providers map[string]exchange.Provider
	log       *zap.Logger
}

func NewExchange(providers []exchange.Provider, log *zap.Logger) *Exchange {
	registered := make(map[string]exchange.Provider, len(providers))
	for _, provider := range providers {
		registered[provider.Name()] = provider
	}
	return &Exchange{providers: registered, log: log}
}

func (h *Exchange) Balances(c *gin.Context) {
	name := c.Param("provider")
	provider, ok := h.providers[name]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "provider_not_found", "message": "exchange provider not found"},
		})
		return
	}

	balances, err := provider.Balances(c.Request.Context())
	if err != nil {
		h.log.Warn("get exchange balances failed", zap.String("provider", name), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"code": "upstream_error", "message": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "balances": balances})
}
