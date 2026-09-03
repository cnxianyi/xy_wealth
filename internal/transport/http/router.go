package httptransport

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/xy-wealth/xy-wealth/internal/config"
	"github.com/xy-wealth/xy-wealth/internal/modules/exchange"
	"github.com/xy-wealth/xy-wealth/internal/modules/summary"
	"github.com/xy-wealth/xy-wealth/internal/transport/http/handler"
	appmiddleware "github.com/xy-wealth/xy-wealth/internal/transport/http/middleware"
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

	health := router.Group("/health")
	health.GET("/live", healthHandler.Live)
	health.GET("/ready", healthHandler.Ready)

	v1 := router.Group("/api/v1")
	v1.GET("/exchanges/:provider/balances", exchangeHandler.Balances)
	v1.GET("/summary", summaryHandler.All)
	v1.GET("/summary/exchanges", summaryHandler.Exchanges)
	v1.GET("/summary/banks", summaryHandler.Banks)

	return router
}
