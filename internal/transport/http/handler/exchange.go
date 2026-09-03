package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Exchange struct {
	providers             map[string]exchange.Provider
	spotProviders         map[string]exchange.SpotProvider
	futuresProviders      map[string]exchange.USDSMFuturesProvider
	positionProviders     map[string]exchange.ContractPositionProvider
	coinMFuturesProviders map[string]exchange.COINMFuturesProvider
	log                   *zap.Logger
}

func NewExchange(providers []exchange.Provider, log *zap.Logger) *Exchange {
	registered := make(map[string]exchange.Provider, len(providers))
	spotProviders := make(map[string]exchange.SpotProvider, len(providers))
	futuresProviders := make(map[string]exchange.USDSMFuturesProvider, len(providers))
	positionProviders := make(map[string]exchange.ContractPositionProvider, len(providers))
	coinMFuturesProviders := make(map[string]exchange.COINMFuturesProvider, len(providers))
	for _, provider := range providers {
		registered[provider.Name()] = provider
		if spotProvider, ok := provider.(exchange.SpotProvider); ok {
			spotProviders[provider.Name()] = spotProvider
		}
		if futuresProvider, ok := provider.(exchange.USDSMFuturesProvider); ok {
			futuresProviders[provider.Name()] = futuresProvider
		}
		if positionProvider, ok := provider.(exchange.ContractPositionProvider); ok {
			positionProviders[provider.Name()] = positionProvider
		}
		if coinMFuturesProvider, ok := provider.(exchange.COINMFuturesProvider); ok {
			coinMFuturesProviders[provider.Name()] = coinMFuturesProvider
		}
	}
	return &Exchange{
		providers:             registered,
		spotProviders:         spotProviders,
		futuresProviders:      futuresProviders,
		positionProviders:     positionProviders,
		coinMFuturesProviders: coinMFuturesProviders,
		log:                   log,
	}
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

func (h *Exchange) Ping(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	if err := provider.Ping(c.Request.Context()); err != nil {
		h.upstreamError(c, name, "ping exchange failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "status": "ok"})
}

func (h *Exchange) ServerTime(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	serverTime, err := provider.ServerTime(c.Request.Context())
	if err != nil {
		h.upstreamError(c, name, "get exchange server time failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "server_time": serverTime.ServerTime})
}

func (h *Exchange) ExchangeInfo(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	info, err := provider.ExchangeInfo(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get exchange information failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "info": info})
}

func (h *Exchange) Depth(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	limit, err := parseQueryInt(c, "limit", 0)
	if err != nil {
		h.handleSpotError(c, name, "get order book failed", err)
		return
	}
	orderBook, err := provider.Depth(c.Request.Context(), c.Query("symbol"), limit)
	if err != nil {
		h.handleSpotError(c, name, "get order book failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "order_book": orderBook})
}

func (h *Exchange) Klines(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	limit, err := parseQueryInt(c, "limit", 0)
	if err != nil {
		h.handleSpotError(c, name, "get klines failed", err)
		return
	}
	startTime, err := parseOptionalQueryInt64(c, "startTime")
	if err != nil {
		h.handleSpotError(c, name, "get klines failed", err)
		return
	}
	endTime, err := parseOptionalQueryInt64(c, "endTime")
	if err != nil {
		h.handleSpotError(c, name, "get klines failed", err)
		return
	}
	klines, err := provider.Klines(c.Request.Context(), exchange.KlinesRequest{
		Symbol:    c.Query("symbol"),
		Interval:  c.Query("interval"),
		StartTime: startTime,
		EndTime:   endTime,
		TimeZone:  c.Query("timeZone"),
		Limit:     limit,
	})
	if err != nil {
		h.handleSpotError(c, name, "get klines failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "klines": klines})
}

func (h *Exchange) Ticker24hr(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.Ticker24hr(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get 24-hour ticker failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) TickerPrice(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.TickerPrice(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get ticker price failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) BookTicker(c *gin.Context) {
	name, provider, ok := h.spotProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.BookTicker(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get book ticker failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) FuturesPing(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	if err := provider.FuturesPing(c.Request.Context()); err != nil {
		h.upstreamError(c, name, "ping USDⓈ-M Futures exchange failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "status": "ok"})
}

func (h *Exchange) FuturesServerTime(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	serverTime, err := provider.FuturesServerTime(c.Request.Context())
	if err != nil {
		h.upstreamError(c, name, "get USDⓈ-M Futures server time failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "server_time": serverTime.ServerTime})
}

func (h *Exchange) FuturesExchangeInfo(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	info, err := provider.FuturesExchangeInfo(c.Request.Context())
	if err != nil {
		h.upstreamError(c, name, "get USDⓈ-M Futures exchange information failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "info": info})
}

func (h *Exchange) FuturesDepth(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	limit, err := parseQueryInt(c, "limit", 0)
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures order book failed", err)
		return
	}
	orderBook, err := provider.FuturesDepth(c.Request.Context(), c.Query("symbol"), limit)
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures order book failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "order_book": orderBook})
}

func (h *Exchange) FuturesKlines(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	limit, err := parseQueryInt(c, "limit", 0)
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures klines failed", err)
		return
	}
	startTime, err := parseOptionalQueryInt64(c, "startTime")
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures klines failed", err)
		return
	}
	endTime, err := parseOptionalQueryInt64(c, "endTime")
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures klines failed", err)
		return
	}
	klines, err := provider.FuturesKlines(c.Request.Context(), exchange.KlinesRequest{
		Symbol:    c.Query("symbol"),
		Interval:  c.Query("interval"),
		StartTime: startTime,
		EndTime:   endTime,
		TimeZone:  c.Query("timeZone"),
		Limit:     limit,
	})
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures klines failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "klines": klines})
}

func (h *Exchange) FuturesTicker24hr(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.FuturesTicker24hr(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures 24-hour ticker failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) FuturesTickerPrice(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.FuturesTickerPrice(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures ticker price failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) FuturesBookTicker(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.FuturesBookTicker(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures book ticker failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) FuturesPremiumIndex(c *gin.Context) {
	name, provider, ok := h.futuresProvider(c)
	if !ok {
		return
	}
	premiumIndex, err := provider.FuturesPremiumIndex(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get USDⓈ-M Futures premium index failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "premium_index": premiumIndex})
}

func (h *Exchange) ContractPositions(c *gin.Context) {
	name, provider, ok := h.contractPositionProvider(c)
	if !ok {
		return
	}
	positions, err := provider.ContractPositions(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get contract positions failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "positions": positions})
}

func (h *Exchange) CoinMFuturesPing(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	if err := provider.CoinMFuturesPing(c.Request.Context()); err != nil {
		h.upstreamError(c, name, "ping COIN-M Futures exchange failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "status": "ok"})
}

func (h *Exchange) CoinMFuturesServerTime(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	serverTime, err := provider.CoinMFuturesServerTime(c.Request.Context())
	if err != nil {
		h.upstreamError(c, name, "get COIN-M Futures server time failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "server_time": serverTime.ServerTime})
}

func (h *Exchange) CoinMFuturesExchangeInfo(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	info, err := provider.CoinMFuturesExchangeInfo(c.Request.Context())
	if err != nil {
		h.upstreamError(c, name, "get COIN-M Futures exchange information failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "info": info})
}

func (h *Exchange) CoinMFuturesDepth(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	limit, err := parseQueryInt(c, "limit", 0)
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures order book failed", err)
		return
	}
	orderBook, err := provider.CoinMFuturesDepth(c.Request.Context(), c.Query("symbol"), limit)
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures order book failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "order_book": orderBook})
}

func (h *Exchange) CoinMFuturesKlines(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	limit, err := parseQueryInt(c, "limit", 0)
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures klines failed", err)
		return
	}
	startTime, err := parseOptionalQueryInt64(c, "startTime")
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures klines failed", err)
		return
	}
	endTime, err := parseOptionalQueryInt64(c, "endTime")
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures klines failed", err)
		return
	}
	klines, err := provider.CoinMFuturesKlines(c.Request.Context(), exchange.KlinesRequest{
		Symbol:    c.Query("symbol"),
		Interval:  c.Query("interval"),
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
	})
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures klines failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "klines": klines})
}

func (h *Exchange) CoinMFuturesTicker24hr(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.CoinMFuturesTicker24hr(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures 24-hour ticker failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) CoinMFuturesTickerPrice(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.CoinMFuturesTickerPrice(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures ticker price failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) CoinMFuturesBookTicker(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	ticker, err := provider.CoinMFuturesBookTicker(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures book ticker failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "ticker": ticker})
}

func (h *Exchange) CoinMFuturesPremiumIndex(c *gin.Context) {
	name, provider, ok := h.coinMFuturesProvider(c)
	if !ok {
		return
	}
	premiumIndex, err := provider.CoinMFuturesPremiumIndex(c.Request.Context(), c.Query("symbol"))
	if err != nil {
		h.handleSpotError(c, name, "get COIN-M Futures premium index failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": name, "premium_index": premiumIndex})
}

func (h *Exchange) spotProvider(c *gin.Context) (string, exchange.SpotProvider, bool) {
	name := c.Param("provider")
	provider, ok := h.spotProviders[name]
	if ok {
		return name, provider, true
	}
	if _, registered := h.providers[name]; registered {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": gin.H{"code": "endpoint_not_supported", "message": "spot endpoint is not supported by this provider"},
		})
		return "", nil, false
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{"code": "provider_not_found", "message": "exchange provider not found"},
	})
	return "", nil, false
}

func (h *Exchange) futuresProvider(c *gin.Context) (string, exchange.USDSMFuturesProvider, bool) {
	name := c.Param("provider")
	provider, ok := h.futuresProviders[name]
	if ok {
		return name, provider, true
	}
	if _, registered := h.providers[name]; registered {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": gin.H{"code": "endpoint_not_supported", "message": "USDⓈ-M Futures endpoint is not supported by this provider"},
		})
		return "", nil, false
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{"code": "provider_not_found", "message": "exchange provider not found"},
	})
	return "", nil, false
}

func (h *Exchange) contractPositionProvider(c *gin.Context) (string, exchange.ContractPositionProvider, bool) {
	name := c.Param("provider")
	provider, ok := h.positionProviders[name]
	if ok {
		return name, provider, true
	}
	if _, registered := h.providers[name]; registered {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": gin.H{"code": "endpoint_not_supported", "message": "contract position endpoint is not supported by this provider"},
		})
		return "", nil, false
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{"code": "provider_not_found", "message": "exchange provider not found"},
	})
	return "", nil, false
}

func (h *Exchange) coinMFuturesProvider(c *gin.Context) (string, exchange.COINMFuturesProvider, bool) {
	name := c.Param("provider")
	provider, ok := h.coinMFuturesProviders[name]
	if ok {
		return name, provider, true
	}
	if _, registered := h.providers[name]; registered {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": gin.H{"code": "endpoint_not_supported", "message": "COIN-M Futures endpoint is not supported by this provider"},
		})
		return "", nil, false
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{"code": "provider_not_found", "message": "exchange provider not found"},
	})
	return "", nil, false
}

func (h *Exchange) handleSpotError(c *gin.Context, provider, operation string, err error) {
	var parameterError exchange.InvalidParameterError
	if errors.As(err, &parameterError) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":      "invalid_parameter",
				"message":   parameterError.Error(),
				"parameter": parameterError.Parameter,
			},
		})
		return
	}
	h.upstreamError(c, provider, operation, err)
}

func (h *Exchange) upstreamError(c *gin.Context, provider, operation string, err error) {
	h.log.Warn(operation, zap.String("provider", provider), zap.Error(err))
	c.JSON(http.StatusBadGateway, gin.H{
		"error": gin.H{"code": "upstream_error", "message": err.Error()},
	})
}

func parseQueryInt(c *gin.Context, name string, defaultValue int) (int, error) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, exchange.InvalidParameterError{Parameter: name, Message: "must be an integer"}
	}
	return parsed, nil
}

func parseOptionalQueryInt64(c *gin.Context, name string) (*int64, error) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, exchange.InvalidParameterError{Parameter: name, Message: "must be an integer"}
	}
	return &parsed, nil
}
