package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	idempotencyPrefix = "idem:orders:"

	lockTTL   = 30 * time.Second
	resultTTL = 24 * time.Hour
)

type IdempotencyStore struct {
	client *redis.Client
}

func NewIdempotencyStore(client *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{client: client}
}

func (s *IdempotencyStore) Acquire(ctx context.Context, key, value string) (string, bool, error) {
	acquired, err := s.client.SetNX(ctx, idempotencyKey(key), value, lockTTL).Result()
	if err != nil {
		return "", false, err
	}
	if acquired {
		return "", true, nil
	}

	stored, err := s.client.Get(ctx, idempotencyKey(key)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return stored, false, nil
}

func (s *IdempotencyStore) Save(ctx context.Context, key, value string) error {
	return s.client.Set(ctx, idempotencyKey(key), value, resultTTL).Err()
}

func (s *IdempotencyStore) Release(ctx context.Context, key string) error {
	return s.client.Del(ctx, idempotencyKey(key)).Err()
}

func idempotencyKey(key string) string {
	return idempotencyPrefix + key
}
