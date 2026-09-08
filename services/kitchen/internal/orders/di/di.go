package di

import (
	"context"
	"net"
	"os"

	"avito-kitchen/pkg/config"
	"avito-kitchen/pkg/repo"
	"avito-kitchen/services/kitchen/internal/orders/decisions"
	"avito-kitchen/services/kitchen/internal/orders/domain"
	"avito-kitchen/services/kitchen/internal/orders/infra/postgres"
	ordersredis "avito-kitchen/services/kitchen/internal/orders/infra/redis"
	"avito-kitchen/services/kitchen/internal/orders/infra/webhook"
	"avito-kitchen/services/kitchen/internal/orders/service"
	ordershttp "avito-kitchen/services/kitchen/internal/orders/transport/http"

	"github.com/google/uuid"
	"go.uber.org/fx"
)

var (
	_ service.Repository       = (*postgres.Repo)(nil)
	_ service.UserRepository   = (*postgres.Repo)(nil)
	_ service.RatingRepository = (*postgres.Repo)(nil)
	_ decisions.Repository     = (*postgres.Repo)(nil)
	_ service.Notifier         = (*decisions.Processor)(nil)
)

var Module = fx.Module("orders",
	fx.Provide(
		repo.NewCtxGetter,
		newInstanceID,
		fx.Annotate(
			postgres.NewRepo,
			fx.As(new(service.Repository)),
			fx.As(new(service.UserRepository)),
			fx.As(new(service.RatingRepository)),
			fx.As(new(decisions.Repository)),
		),
		fx.Annotate(
			ordersredis.NewEventStream,
			fx.As(new(service.EventStream)),
			fx.As(new(decisions.EventPublisher)),
		),
		fx.Annotate(
			ordersredis.NewIdempotencyStore,
			fx.As(new(service.IdempotencyStore)),
		),
		fx.Annotate(
			webhook.NewClient,
			fx.As(new(decisions.WebhookClient)),
		),
		decisions.New,
		func(p *decisions.Processor) service.Notifier { return p },
		service.New,
		ordershttp.NewHandler,
	),
	fx.Invoke(
		(*ordershttp.Handler).Register,
		runProcessor,
	),
)

func newInstanceID(conf config.ServConfig) domain.InstanceID {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = uuid.NewString()
	}
	return domain.InstanceID(net.JoinHostPort(host, conf.Port))
}

func runProcessor(lc fx.Lifecycle, processor *decisions.Processor) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			processor.Start()
			return nil
		},
		OnStop: processor.Stop,
	})
}
