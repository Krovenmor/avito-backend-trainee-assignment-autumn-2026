package redis

import (
	"context"
	"errors"
	"time"

	"avito-kitchen/services/kitchen/internal/orders/domain"

	"github.com/redis/go-redis/v9"
)

const (
	streamPrefix = "order:"
	streamSuffix = ":events"

	streamMaxLen = 10
	streamTTL    = 24 * time.Hour
	streamBlock  = 5 * time.Second

	fieldStatus = "status"
	fieldUserID = "user_id"
	fieldSource = "source"
)

type EventStream struct {
	client *redis.Client
}

func NewEventStream(client *redis.Client) *EventStream {
	return &EventStream{client: client}
}

func (s *EventStream) Publish(ctx context.Context, ev domain.OrderEvent) error {
	key := streamKey(ev.OrderID)

	err := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		MaxLen: streamMaxLen,
		Approx: true,
		Values: map[string]any{
			fieldStatus: ev.Status,
			fieldUserID: ev.UserID,
			fieldSource: string(ev.Source),
		},
	}).Err()
	if err != nil {
		return err
	}

	return s.client.Expire(ctx, key, streamTTL).Err()
}

func (s *EventStream) Subscribe(ctx context.Context, orderID string) (<-chan domain.OrderEvent, error) {
	key := streamKey(orderID)

	history, err := s.client.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, domain.ErrNotFound
	}

	out := make(chan domain.OrderEvent)
	go s.stream(ctx, key, orderID, history, out)

	return out, nil
}

func (s *EventStream) stream(ctx context.Context, key, orderID string, history []redis.XMessage, out chan<- domain.OrderEvent) {
	defer close(out)

	lastID := "0"
	for _, message := range history {
		if !send(ctx, out, toOrderEvent(orderID, message)) {
			return
		}
		lastID = message.ID
	}

	for {
		streams, err := s.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{key, lastID},
			Block:   streamBlock,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				if !send(ctx, out, toOrderEvent(orderID, message)) {
					return
				}
				lastID = message.ID
			}
		}
	}
}

func send(ctx context.Context, out chan<- domain.OrderEvent, ev domain.OrderEvent) bool {
	select {
	case out <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

func toOrderEvent(orderID string, message redis.XMessage) domain.OrderEvent {
	event := domain.OrderEvent{OrderID: orderID, Source: domain.SourcePlatform}
	if status, ok := message.Values[fieldStatus].(string); ok {
		event.Status = status
	}
	if userID, ok := message.Values[fieldUserID].(string); ok {
		event.UserID = userID
	}
	if source, ok := message.Values[fieldSource].(string); ok {
		event.Source = domain.EventSource(source)
	}
	return event
}

func streamKey(orderID string) string {
	return streamPrefix + orderID + streamSuffix
}
