package service

import (
	"context"

	"avito-kitchen/services/kitchen/internal/orders/domain"
)

type Service interface {
	CreateOrder(ctx context.Context, in domain.CreateOrderInput) (*domain.CreateOrderOutput, error)
	CancelOrder(ctx context.Context, orderID, userID string) error
	MarkDelivered(ctx context.Context, orderID, placeID string) error
	AddTracking(ctx context.Context, in domain.AddTrackingInput) error
	GetOrder(ctx context.Context, orderID, userID string) (*domain.Order, error)
	ListOrderTracking(ctx context.Context, orderID, userID string) ([]domain.TrackingEvent, error)
	ListUserOrders(ctx context.Context, p domain.ListUserOrdersParams) ([]domain.OrderSummary, error)
	SubscribeOrderEvents(ctx context.Context, orderID, userID string) (<-chan domain.OrderEvent, error)
	RateOrder(ctx context.Context, in domain.RateOrderInput) error
}

type Notifier interface {
	Notify(order domain.PendingOrder)
}
