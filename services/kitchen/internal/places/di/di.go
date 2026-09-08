package di

import (
	"avito-kitchen/services/kitchen/internal/places/infra/postgres"
	"avito-kitchen/services/kitchen/internal/places/service"
	placeshttp "avito-kitchen/services/kitchen/internal/places/transport/http"

	"go.uber.org/fx"
)

var _ service.Repository = (*postgres.Repo)(nil)

var Module = fx.Module("places",
	fx.Provide(
		fx.Annotate(
			postgres.NewRepo,
			fx.As(new(service.Repository)),
		),
		service.New,
		placeshttp.NewHandler,
	),
	fx.Invoke((*placeshttp.Handler).Register),
)
