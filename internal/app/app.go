package app

import (
	"context"
	"fmt"

	"github.com/cnxianyi/xy_wealth/internal/config"
	authmodule "github.com/cnxianyi/xy_wealth/internal/modules/auth"
	"github.com/cnxianyi/xy_wealth/internal/modules/bank"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange/binance"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange/bitget"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange/weex"
	"github.com/cnxianyi/xy_wealth/internal/modules/summary"
	"github.com/cnxianyi/xy_wealth/internal/platform/database"
	httptransport "github.com/cnxianyi/xy_wealth/internal/transport/http"
	"go.uber.org/zap"
)

func Run(ctx context.Context, cfg config.Config, log *zap.Logger) error {
	postgres, err := database.OpenPostgres(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer postgres.Close()

	redisClient, err := database.OpenRedis(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	exchangeProviders := []exchange.Provider{
		binance.New(cfg.Binance),
		bitget.New(cfg.Bitget),
		weex.New(cfg.Weex),
	}
	bankProviders := []bank.Provider{}
	summaryService := summary.New(exchangeProviders, bankProviders)
	authService := authmodule.New(cfg.Auth, redisClient)

	router := httptransport.NewRouter(cfg, log, postgres, redisClient, exchangeProviders, summaryService, authService)
	server := httptransport.NewServer(cfg.HTTP, router, log)

	log.Info("application initialized",
		zap.String("name", cfg.App.Name),
		zap.String("environment", cfg.App.Environment),
	)
	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("run server: %w", err)
	}
	return nil
}
