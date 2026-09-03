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

func TestExchangeSpotHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := &stubSpotProvider{}
	handler := NewExchange([]exchange.Provider{provider}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/spot/ping", handler.Ping)
	router.GET("/exchanges/:provider/spot/time", handler.ServerTime)
	router.GET("/exchanges/:provider/spot/exchange-info", handler.ExchangeInfo)
	router.GET("/exchanges/:provider/spot/depth", handler.Depth)
	router.GET("/exchanges/:provider/spot/klines", handler.Klines)
	router.GET("/exchanges/:provider/spot/ticker/24hr", handler.Ticker24hr)
	router.GET("/exchanges/:provider/spot/ticker/price", handler.TickerPrice)
	router.GET("/exchanges/:provider/spot/ticker/book", handler.BookTicker)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "ping", path: "/exchanges/binance/spot/ping", want: `"status":"ok"`},
		{name: "time", path: "/exchanges/binance/spot/time", want: `"server_time":1700000000123`},
		{name: "exchange info", path: "/exchanges/binance/spot/exchange-info?symbol=BTCUSDT", want: `"symbol":"BTCUSDT"`},
		{name: "depth", path: "/exchanges/binance/spot/depth?symbol=BTCUSDT&limit=20", want: `"last_update_id":123`},
		{name: "klines", path: "/exchanges/binance/spot/klines?symbol=BTCUSDT&interval=1m", want: `"open":"100.00"`},
		{name: "24hr ticker", path: "/exchanges/binance/spot/ticker/24hr?symbol=BTCUSDT", want: `"last_price":"1000.00"`},
		{name: "price ticker", path: "/exchanges/binance/spot/ticker/price?symbol=BTCUSDT", want: `"price":"1000.00"`},
		{name: "book ticker", path: "/exchanges/binance/spot/ticker/book?symbol=BTCUSDT", want: `"ask_price":"1000.10"`},
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

func TestExchangeDepthHandlerValidatesLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewExchange([]exchange.Provider{&stubSpotProvider{}}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/spot/depth", handler.Depth)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/exchanges/binance/spot/depth?symbol=BTCUSDT&limit=not-a-number", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"code":"invalid_parameter"`)) {
		t.Fatalf("response = %s", response.Body.String())
	}
}

func TestExchangeUSDSMFuturesHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := &stubSpotProvider{}
	handler := NewExchange([]exchange.Provider{provider}, zap.NewNop())
	router := gin.New()
	router.GET("/exchanges/:provider/futures/usdm/ping", handler.FuturesPing)
	router.GET("/exchanges/:provider/futures/usdm/time", handler.FuturesServerTime)
	router.GET("/exchanges/:provider/futures/usdm/exchange-info", handler.FuturesExchangeInfo)
	router.GET("/exchanges/:provider/futures/usdm/depth", handler.FuturesDepth)
	router.GET("/exchanges/:provider/futures/usdm/klines", handler.FuturesKlines)
	router.GET("/exchanges/:provider/futures/usdm/ticker/24hr", handler.FuturesTicker24hr)
	router.GET("/exchanges/:provider/futures/usdm/ticker/price", handler.FuturesTickerPrice)
	router.GET("/exchanges/:provider/futures/usdm/ticker/book", handler.FuturesBookTicker)
	router.GET("/exchanges/:provider/futures/usdm/premium-index", handler.FuturesPremiumIndex)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "ping", path: "/exchanges/binance/futures/usdm/ping", want: `"status":"ok"`},
		{name: "time", path: "/exchanges/binance/futures/usdm/time", want: `"server_time":1700000000123`},
		{name: "exchange info", path: "/exchanges/binance/futures/usdm/exchange-info", want: `"contract_type":"PERPETUAL"`},
		{name: "depth", path: "/exchanges/binance/futures/usdm/depth?symbol=BTCUSDT&limit=20", want: `"last_update_id":123`},
		{name: "klines", path: "/exchanges/binance/futures/usdm/klines?symbol=BTCUSDT&interval=1m", want: `"close":"100.50"`},
		{name: "24hr ticker", path: "/exchanges/binance/futures/usdm/ticker/24hr?symbol=BTCUSDT", want: `"last_price":"1000.00"`},
		{name: "price ticker", path: "/exchanges/binance/futures/usdm/ticker/price?symbol=BTCUSDT", want: `"price":"1000.00"`},
		{name: "book ticker", path: "/exchanges/binance/futures/usdm/ticker/book?symbol=BTCUSDT", want: `"ask_price":"1000.10"`},
		{name: "premium index", path: "/exchanges/binance/futures/usdm/premium-index?symbol=BTCUSDT", want: `"mark_price":"1000.00"`},
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

type stubSpotProvider struct{}

func (stubSpotProvider) Name() string { return "binance" }
func (stubSpotProvider) Balances(context.Context) ([]asset.Balance, error) {
	return nil, errors.New("not used")
}
func (stubSpotProvider) Ping(context.Context) error { return nil }
func (stubSpotProvider) ServerTime(context.Context) (exchange.ServerTime, error) {
	return exchange.ServerTime{ServerTime: 1700000000123}, nil
}
func (stubSpotProvider) ExchangeInfo(context.Context, string) (exchange.ExchangeInfo, error) {
	return exchange.ExchangeInfo{Symbols: []exchange.SymbolInfo{{Symbol: "BTCUSDT"}}}, nil
}
func (stubSpotProvider) Depth(context.Context, string, int) (exchange.OrderBook, error) {
	return exchange.OrderBook{LastUpdateID: 123}, nil
}
func (stubSpotProvider) Klines(context.Context, exchange.KlinesRequest) ([]exchange.Kline, error) {
	return []exchange.Kline{{Open: "100.00"}}, nil
}
func (stubSpotProvider) Ticker24hr(context.Context, string) (exchange.Ticker24hr, error) {
	return exchange.Ticker24hr{LastPrice: "1000.00"}, nil
}
func (stubSpotProvider) TickerPrice(context.Context, string) (exchange.PriceTicker, error) {
	return exchange.PriceTicker{Price: "1000.00"}, nil
}
func (stubSpotProvider) BookTicker(context.Context, string) (exchange.BookTicker, error) {
	return exchange.BookTicker{AskPrice: "1000.10"}, nil
}

func (stubSpotProvider) FuturesPing(context.Context) error { return nil }
func (stubSpotProvider) FuturesServerTime(context.Context) (exchange.ServerTime, error) {
	return exchange.ServerTime{ServerTime: 1700000000123}, nil
}
func (stubSpotProvider) FuturesExchangeInfo(context.Context) (exchange.USDSMFuturesExchangeInfo, error) {
	return exchange.USDSMFuturesExchangeInfo{
		Symbols: []exchange.USDSMFuturesSymbolInfo{{ContractType: "PERPETUAL"}},
	}, nil
}
func (stubSpotProvider) FuturesDepth(context.Context, string, int) (exchange.FuturesOrderBook, error) {
	return exchange.FuturesOrderBook{LastUpdateID: 123}, nil
}
func (stubSpotProvider) FuturesKlines(context.Context, exchange.KlinesRequest) ([]exchange.Kline, error) {
	return []exchange.Kline{{Close: "100.50"}}, nil
}
func (stubSpotProvider) FuturesTicker24hr(context.Context, string) (exchange.FuturesTicker24hr, error) {
	return exchange.FuturesTicker24hr{LastPrice: "1000.00"}, nil
}
func (stubSpotProvider) FuturesTickerPrice(context.Context, string) (exchange.PriceTicker, error) {
	return exchange.PriceTicker{Price: "1000.00"}, nil
}
func (stubSpotProvider) FuturesBookTicker(context.Context, string) (exchange.BookTicker, error) {
	return exchange.BookTicker{AskPrice: "1000.10"}, nil
}
func (stubSpotProvider) FuturesPremiumIndex(context.Context, string) (exchange.FuturesPremiumIndex, error) {
	return exchange.FuturesPremiumIndex{MarkPrice: "1000.00"}, nil
}
