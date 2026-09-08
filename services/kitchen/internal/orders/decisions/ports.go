package decisions

import (
	"context"

	"avito-kitchen/services/kitchen/internal/orders/domain"
)

type WebhookClient interface {
	CallOnce(ctx context.Context, webhookURL string, order domain.Order) (domain.Decision, bool)
}

type EventPublisher interface {
	Publish(ctx context.Context, ev domain.OrderEvent) error
}
