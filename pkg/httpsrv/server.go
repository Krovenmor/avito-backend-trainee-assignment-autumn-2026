package httpsrv

import (
	"context"
	"errors"
	"net/http"
	"time"

	"avito-kitchen/pkg/config"
	"avito-kitchen/pkg/health"
	"avito-kitchen/pkg/httpmw"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 2 * time.Minute
)

func New(conf config.ServConfig, mux *http.ServeMux, log *zap.Logger) *http.Server {
	return &http.Server{
		Addr:              conf.Address(),
		Handler:           httpmw.CORS(httpmw.Logging(log, health.LivenessPath, health.ReadinessPath)(mux)),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func Run(lc fx.Lifecycle, srv *http.Server, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error("http server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
