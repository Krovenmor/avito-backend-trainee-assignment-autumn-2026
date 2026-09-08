package di

import (
	"context"
	"errors"
	"net/http"
	"time"

	"avito-kitchen/pkg/config"
	"avito-kitchen/pkg/health"
	"avito-kitchen/pkg/httpsrv"
	"avito-kitchen/pkg/logger"
	"avito-kitchen/services/testapi/internal/client"
	testapiconfig "avito-kitchen/services/testapi/internal/config"
	"avito-kitchen/services/testapi/internal/stock"
	testapihttp "avito-kitchen/services/testapi/internal/transport/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	registerTimeout  = 60 * time.Second
	registerAttempt  = 10 * time.Second
	registerInterval = 2 * time.Second

	offlineSalesQuantity = 2

	restockTimeout = 10 * time.Second
)

var Module = fx.Options(
	fx.Provide(
		config.GetServConfig,
		config.GetLoggerConfig,
		testapiconfig.GetConfig,

		logger.NewLogger,
		client.New,
		client.NewCredentials,
		newStock,

		http.NewServeMux,
		testapihttp.NewHandler,
		httpsrv.New,
	),
	fx.Invoke(
		(*testapihttp.Handler).Register,
		registerHealth,
		joinPlatform,
		sellOffline,
		restock,
		httpsrv.Run,
	),
)

func restock(
	lc fx.Lifecycle,
	apiClient *client.Client,
	credentials *client.Credentials,
	inventory *stock.Stock,
	conf testapiconfig.Config,
	log *zap.Logger,
) {
	if conf.RestockInterval <= 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				defer close(done)

				ticker := time.NewTicker(conf.RestockInterval)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						externalID, quantity, restocked := inventory.Restock()
						if !restocked {
							continue
						}

						placeID, apiKey := credentials.Get()
						if apiKey == "" {
							continue
						}

						callCtx, callCancel := context.WithTimeout(ctx, restockTimeout)
						err := apiClient.Restock(callCtx, placeID, externalID, apiKey, quantity)
						callCancel()
						if err != nil {
							log.Warn("restock", zap.String("external_id", externalID), zap.Error(err))
							continue
						}

						log.Info("restocked",
							zap.String("external_id", externalID), zap.Int32("quantity", quantity))
					case <-ctx.Done():
						return
					}
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			<-done
			return nil
		},
	})
}

func registerHealth(mux *http.ServeMux) {
	health.Register(mux, nil)
}

func newStock() *stock.Stock {
	return stock.New(menu)
}

func joinPlatform(
	lc fx.Lifecycle,
	apiClient *client.Client,
	credentials *client.Credentials,
	inventory *stock.Stock,
	conf testapiconfig.Config,
	log *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx, cancel := context.WithTimeout(context.Background(), registerTimeout)
			defer cancel()

			place, err := registerWithRetry(ctx, apiClient, conf, log)
			if err != nil {
				return err
			}
			credentials.Set(place.PlaceID, place.APIKey)

			if err := apiClient.AddMenu(ctx, place.PlaceID, place.APIKey, inventory.Items()); err != nil {
				return err
			}

			log.Info("joined the platform",
				zap.String("place_id", place.PlaceID), zap.Int("menu_items", len(inventory.Items())))
			return nil
		},
	})
}

func registerWithRetry(
	ctx context.Context,
	apiClient *client.Client,
	conf testapiconfig.Config,
	log *zap.Logger,
) (*client.RegisterPlaceResponse, error) {
	request := placeRegistration(conf.PublicURL + testapihttp.WebhookPath)

	for {
		attemptCtx, cancel := context.WithTimeout(ctx, registerAttempt)
		place, err := apiClient.RegisterPlace(attemptCtx, request)
		cancel()

		if err == nil {
			return place, nil
		}
		log.Warn("register place failed, retrying", zap.Error(err))

		select {
		case <-time.After(registerInterval):
		case <-ctx.Done():
			return nil, errors.Join(ctx.Err(), err)
		}
	}
}

func sellOffline(lc fx.Lifecycle, inventory *stock.Stock, conf testapiconfig.Config, log *zap.Logger) {
	if conf.OfflineSalesInterval <= 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				defer close(done)

				ticker := time.NewTicker(conf.OfflineSalesInterval)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						if externalID, quantity, sold := inventory.SellOffline(offlineSalesQuantity); sold {
							log.Info("offline sale",
								zap.String("external_id", externalID), zap.Int32("quantity", quantity))
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			<-done
			return nil
		},
	})
}
