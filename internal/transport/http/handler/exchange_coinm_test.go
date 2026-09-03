package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestExchangeCOINMFuturesHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewExchange([]exchange.Provider{stubCOINMFuturesProvider{}}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/futures/coinm/ping", handler.CoinMFuturesPing)
	router.GET("/exchanges/:provider/futures/coinm/time", handler.CoinMFuturesServerTime)
	router.GET("/exchanges/:provider/futures/coinm/exchange-info", handler.CoinMFuturesExchangeInfo)
	router.GET("/exchanges/:provider/futures/coinm/depth", handler.CoinMFuturesDepth)
	router.GET("/exchanges/:provider/futures/coinm/klines", handler.CoinMFuturesKlines)
	router.GET("/exchanges/:provider/futures/coinm/ticker/24hr", handler.CoinMFuturesTicker24hr)
	router.GET("/exchanges/:provider/futures/coinm/ticker/price", handler.CoinMFuturesTickerPrice)
	router.GET("/exchanges/:provider/futures/coinm/ticker/book", handler.CoinMFuturesBookTicker)
	router.GET("/exchanges/:provider/futures/coinm/premium-index", handler.CoinMFuturesPremiumIndex)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "ping", path: "/exchanges/binance/futures/coinm/ping", want: `"status":"ok"`},
		{name: "time", path: "/exchanges/binance/futures/coinm/time", want: `"server_time":1700000000123`},
		{name: "exchange info", path: "/exchanges/binance/futures/coinm/exchange-info", want: `"contract_status":"TRADING"`},
		{name: "depth", path: "/exchanges/binance/futures/coinm/depth?symbol=BTCUSD_PERP&limit=20", want: `"last_update_id":123`},
		{name: "klines", path: "/exchanges/binance/futures/coinm/klines?symbol=BTCUSD_PERP&interval=1m", want: `"close":"100.50"`},
		{name: "24hr ticker", path: "/exchanges/binance/futures/coinm/ticker/24hr?symbol=BTCUSD_PERP", want: `"pair":"BTCUSD"`},
		{name: "price ticker", path: "/exchanges/binance/futures/coinm/ticker/price?symbol=BTCUSD_PERP", want: `"price":"1000.00"`},
		{name: "book ticker", path: "/exchanges/binance/futures/coinm/ticker/book?symbol=BTCUSD_PERP", want: `"ask_price":"1000.10"`},
		{name: "premium index", path: "/exchanges/binance/futures/coinm/premium-index?symbol=BTCUSD_PERP", want: `"mark_price":"1000.00"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
			}
			if !bytes.Contains(response.Body.Bytes(), []byte(test.want)) {
				t.Fatalf("response does not contain %q: %s", test.want, response.Body.String())
			}
		})
	}
}

func TestExchangeCOINMFuturesProviderErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewExchange([]exchange.Provider{stubSpotProvider{}}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/futures/coinm/ping", handler.CoinMFuturesPing)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/missing/futures/coinm/ping", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/binance/futures/coinm/ping", nil))
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"code":"endpoint_not_supported"`)) {
		t.Fatalf("response = %s", response.Body.String())
	}
}

type stubCOINMFuturesProvider struct{}

func (stubCOINMFuturesProvider) Name() string { return "binance" }
func (stubCOINMFuturesProvider) Balances(context.Context) ([]asset.Balance, error) {
	return nil, errors.New("not used")
}
func (stubCOINMFuturesProvider) CoinMFuturesPing(context.Context) error { return nil }
func (stubCOINMFuturesProvider) CoinMFuturesServerTime(context.Context) (exchange.ServerTime, error) {
	return exchange.ServerTime{ServerTime: 1700000000123}, nil
}
func (stubCOINMFuturesProvider) CoinMFuturesExchangeInfo(context.Context) (exchange.COINMFuturesExchangeInfo, error) {
	return exchange.COINMFuturesExchangeInfo{
		Symbols: []exchange.COINMFuturesSymbolInfo{{ContractStatus: "TRADING"}},
	}, nil
}
func (stubCOINMFuturesProvider) CoinMFuturesDepth(context.Context, string, int) (exchange.FuturesOrderBook, error) {
	return exchange.FuturesOrderBook{LastUpdateID: 123}, nil
}
func (stubCOINMFuturesProvider) CoinMFuturesKlines(context.Context, exchange.KlinesRequest) ([]exchange.Kline, error) {
	return []exchange.Kline{{Close: "100.50"}}, nil
}
func (stubCOINMFuturesProvider) CoinMFuturesTicker24hr(context.Context, string) (exchange.COINMFuturesTicker24hr, error) {
	return exchange.COINMFuturesTicker24hr{Pair: "BTCUSD"}, nil
}
func (stubCOINMFuturesProvider) CoinMFuturesTickerPrice(context.Context, string) (exchange.COINMFuturesPriceTicker, error) {
	return exchange.COINMFuturesPriceTicker{Price: "1000.00"}, nil
}
func (stubCOINMFuturesProvider) CoinMFuturesBookTicker(context.Context, string) (exchange.COINMFuturesBookTicker, error) {
	return exchange.COINMFuturesBookTicker{AskPrice: "1000.10"}, nil
}
func (stubCOINMFuturesProvider) CoinMFuturesPremiumIndex(context.Context, string) (exchange.COINMFuturesPremiumIndex, error) {
	return exchange.COINMFuturesPremiumIndex{MarkPrice: "1000.00"}, nil
}
