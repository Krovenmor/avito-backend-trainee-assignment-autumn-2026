package di

import (
	"context"
	"net/http"

	"avito-kitchen/pkg/config"
	"avito-kitchen/pkg/health"
	"avito-kitchen/pkg/httpsrv"
	"avito-kitchen/pkg/logger"
	kitchenredis "avito-kitchen/pkg/redis"
	"avito-kitchen/pkg/repo"
	"avito-kitchen/services/kitchen/internal/httpx"
	ordersdi "avito-kitchen/services/kitchen/internal/orders/di"
	placesdi "avito-kitchen/services/kitchen/internal/places/di"
	"avito-kitchen/services/kitchen/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/fx"
)

var Module = fx.Options(
	placesdi.Module,
	ordersdi.Module,

	fx.Provide(
		config.GetRepoConfig,
		config.GetRedisConfig,
		config.GetServConfig,
		config.GetLoggerConfig,

		logger.NewLogger,
		repo.NewPool,
		repo.NewTrManager,
		kitchenredis.NewClient,

		httpx.NewValidator,
		http.NewServeMux,
		httpsrv.New,
	),
	fx.Invoke(
		migrations.MakeMigrations,
		registerSwagger,
		registerHealth,
		httpsrv.Run,
	),
)

func registerHealth(mux *http.ServeMux, pool *pgxpool.Pool, rdb *redis.Client) {
	health.Register(mux, map[string]health.Check{
		"postgres": pool.Ping,
		"redis":    func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
	})
}

func registerSwagger(mux *http.ServeMux) {
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
}
