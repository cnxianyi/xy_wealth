package httptransport

import (
	"database/sql"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/cnxianyi/xy_wealth/internal/modules/summary"
	"github.com/cnxianyi/xy_wealth/internal/transport/http/handler"
	appmiddleware "github.com/cnxianyi/xy_wealth/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewRouter(
	cfg config.Config,
	log *zap.Logger,
	postgres *sql.DB,
	redisClient *redis.Client,
	exchanges []exchange.Provider,
	summaryService *summary.Service,
) *gin.Engine {
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(appmiddleware.RequestID(), appmiddleware.Logger(log), appmiddleware.Recovery(log))

	healthHandler := handler.NewHealth(postgres, redisClient)
	exchangeHandler := handler.NewExchange(exchanges, log)
	summaryHandler := handler.NewSummary(summaryService)
	docsHandler := handler.NewDocs()

	router.GET("/openapi.yaml", docsHandler.OpenAPI)
	router.GET("/docs", docsHandler.UI)

	health := router.Group("/health")
	health.GET("/live", healthHandler.Live)
	health.GET("/ready", healthHandler.Ready)

	v1 := router.Group("/api/v1")
	v1.GET("/exchanges/:provider/balances", exchangeHandler.Balances)
	spot := v1.Group("/exchanges/:provider/spot")
	spot.GET("/ping", exchangeHandler.Ping)
	spot.GET("/time", exchangeHandler.ServerTime)
	spot.GET("/exchange-info", exchangeHandler.ExchangeInfo)
	spot.GET("/depth", exchangeHandler.Depth)
	spot.GET("/klines", exchangeHandler.Klines)
	spot.GET("/ticker/24hr", exchangeHandler.Ticker24hr)
	spot.GET("/ticker/price", exchangeHandler.TickerPrice)
	spot.GET("/ticker/book", exchangeHandler.BookTicker)
	futures := v1.Group("/exchanges/:provider/futures/usdm")
	futures.GET("/ping", exchangeHandler.FuturesPing)
	futures.GET("/time", exchangeHandler.FuturesServerTime)
	futures.GET("/exchange-info", exchangeHandler.FuturesExchangeInfo)
	futures.GET("/depth", exchangeHandler.FuturesDepth)
	futures.GET("/klines", exchangeHandler.FuturesKlines)
	futures.GET("/ticker/24hr", exchangeHandler.FuturesTicker24hr)
	futures.GET("/ticker/price", exchangeHandler.FuturesTickerPrice)
	futures.GET("/ticker/book", exchangeHandler.FuturesBookTicker)
	futures.GET("/premium-index", exchangeHandler.FuturesPremiumIndex)
	futures.GET("/positions", exchangeHandler.ContractPositions)
	futures.GET("/balances", exchangeHandler.ContractBalances)
	coinMFutures := v1.Group("/exchanges/:provider/futures/coinm")
	coinMFutures.GET("/ping", exchangeHandler.CoinMFuturesPing)
	coinMFutures.GET("/time", exchangeHandler.CoinMFuturesServerTime)
	coinMFutures.GET("/exchange-info", exchangeHandler.CoinMFuturesExchangeInfo)
	coinMFutures.GET("/depth", exchangeHandler.CoinMFuturesDepth)
	coinMFutures.GET("/klines", exchangeHandler.CoinMFuturesKlines)
	coinMFutures.GET("/ticker/24hr", exchangeHandler.CoinMFuturesTicker24hr)
	coinMFutures.GET("/ticker/price", exchangeHandler.CoinMFuturesTickerPrice)
	coinMFutures.GET("/ticker/book", exchangeHandler.CoinMFuturesBookTicker)
	coinMFutures.GET("/premium-index", exchangeHandler.CoinMFuturesPremiumIndex)
	v1.GET("/summary", summaryHandler.All)
	v1.GET("/summary/exchanges", summaryHandler.Exchanges)
	v1.GET("/summary/banks", summaryHandler.Banks)

	return router
}
