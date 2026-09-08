package logger

import (
	"context"
	"fmt"

	"avito-kitchen/pkg/config"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(lc fx.Lifecycle, conf config.LoggerConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(conf.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", conf.Level, err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)

	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			_ = l.Sync()
			return nil
		},
	})

	return l, nil
}
