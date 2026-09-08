package redis

import (
	"context"

	"avito-kitchen/pkg/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func NewClient(lc fx.Lifecycle, conf config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: conf.Addr})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return client.Close()
		},
	})

	return client, nil
}
