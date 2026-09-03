package httptransport

import (
	"database/sql"

	"github.com/cnxianyi/xy_wealth/internal/config"
	authmodule "github.com/cnxianyi/xy_wealth/internal/modules/auth"
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
	authService *authmodule.Service,
) *gin.Engine {
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(appmiddleware.RequestID(), appmiddleware.Logger(log), appmiddleware.Recovery(log))

	healthHandler := handler.NewHealth(postgres, redisClient)
	exchangeHandler := handler.NewExchange(exchanges, log)
	summaryHandler := handler.NewSummary(summaryService)
	authHandler := handler.NewAuth(authService, log)
	docsHandler := handler.NewDocs()
	xToken := appmiddleware.XToken(authService, log)

	router.GET("/docs", docsHandler.UI)
	router.GET("/openapi.yaml", xToken, docsHandler.OpenAPI)

	health := router.Group("/health")
	health.GET("/live", healthHandler.Live)
	health.GET("/ready", xToken, healthHandler.Ready)

	v1 := router.Group("/api/v1")
	v1.POST("/auth/login", authHandler.Login)
	protected := v1.Group("")
	protected.Use(xToken)
	protected.GET("/auth/session", authHandler.Session)
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/exchanges/:provider/balances", exchangeHandler.Balances)
	spot := protected.Group("/exchanges/:provider/spot")
	spot.GET("/ping", exchangeHandler.Ping)
	spot.GET("/time", exchangeHandler.ServerTime)
	spot.GET("/exchange-info", exchangeHandler.ExchangeInfo)
	spot.GET("/depth", exchangeHandler.Depth)
	spot.GET("/klines", exchangeHandler.Klines)
	spot.GET("/ticker/24hr", exchangeHandler.Ticker24hr)
	spot.GET("/ticker/price", exchangeHandler.TickerPrice)
	spot.GET("/ticker/book", exchangeHandler.BookTicker)
	futures := protected.Group("/exchanges/:provider/futures/usdm")
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
	futures.GET("/account/balances", exchangeHandler.USDSMFuturesAccountBalances)
	futures.GET("/account/positions", exchangeHandler.USDSMFuturesAccountPositions)
	usdcMFutures := protected.Group("/exchanges/:provider/futures/usdcm")
	usdcMFutures.GET("/ping", exchangeHandler.USDCMFuturesPing)
	usdcMFutures.GET("/time", exchangeHandler.USDCMFuturesServerTime)
	usdcMFutures.GET("/exchange-info", exchangeHandler.USDCMFuturesExchangeInfo)
	usdcMFutures.GET("/depth", exchangeHandler.USDCMFuturesDepth)
	usdcMFutures.GET("/klines", exchangeHandler.USDCMFuturesKlines)
	usdcMFutures.GET("/ticker/24hr", exchangeHandler.USDCMFuturesTicker24hr)
	usdcMFutures.GET("/ticker/price", exchangeHandler.USDCMFuturesTickerPrice)
	usdcMFutures.GET("/ticker/book", exchangeHandler.USDCMFuturesBookTicker)
	usdcMFutures.GET("/premium-index", exchangeHandler.USDCMFuturesPremiumIndex)
	usdcMFutures.GET("/account/balances", exchangeHandler.USDCMFuturesAccountBalances)
	usdcMFutures.GET("/account/positions", exchangeHandler.USDCMFuturesAccountPositions)
	coinMFutures := protected.Group("/exchanges/:provider/futures/coinm")
	coinMFutures.GET("/ping", exchangeHandler.CoinMFuturesPing)
	coinMFutures.GET("/time", exchangeHandler.CoinMFuturesServerTime)
	coinMFutures.GET("/exchange-info", exchangeHandler.CoinMFuturesExchangeInfo)
	coinMFutures.GET("/depth", exchangeHandler.CoinMFuturesDepth)
	coinMFutures.GET("/klines", exchangeHandler.CoinMFuturesKlines)
	coinMFutures.GET("/ticker/24hr", exchangeHandler.CoinMFuturesTicker24hr)
	coinMFutures.GET("/ticker/price", exchangeHandler.CoinMFuturesTickerPrice)
	coinMFutures.GET("/ticker/book", exchangeHandler.CoinMFuturesBookTicker)
	coinMFutures.GET("/premium-index", exchangeHandler.CoinMFuturesPremiumIndex)
	coinMFutures.GET("/account/balances", exchangeHandler.COINMFuturesAccountBalances)
	coinMFutures.GET("/account/positions", exchangeHandler.COINMFuturesAccountPositions)
	protected.GET("/summary", summaryHandler.All)
	protected.GET("/summary/exchanges", summaryHandler.Exchanges)
	protected.GET("/summary/banks", summaryHandler.Banks)

	return router
}
