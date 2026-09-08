package service

import (
	"context"

	"avito-kitchen/services/kitchen/internal/orders/domain"
)

type UserRepository interface {
	UpsertUser(ctx context.Context, userID string) error
}

type RatingRepository interface {
	InsertOrderRating(ctx context.Context, p domain.RateOrderParams) (placeID string, rated bool, err error)
	AddPlaceRating(ctx context.Context, placeID string, score int16) error
}

type Repository interface {
	GetPlaceForOrder(ctx context.Context, placeID string) (*domain.PlaceForOrder, error)
	LockPlaceProducts(ctx context.Context, placeID string, productIDs []string) ([]string, error)
	ReserveStock(ctx context.Context, items []domain.CreateOrderItemInput) ([]domain.UnavailableItem, error)
	InsertOrder(ctx context.Context, p domain.CreateOrderParams) (*domain.OrderTotal, error)

	GetOrderOwner(ctx context.Context, orderID string) (*domain.OrderOwner, error)
	GetOrderStatusForPlace(ctx context.Context, orderID, placeID string) (domain.OrderStatus, error)
	ListUserOrders(ctx context.Context, p domain.ListUserOrdersParams) ([]domain.OrderSummary, error)

	CancelPendingOrder(ctx context.Context, orderID, userID string) (bool, error)
	MarkAcceptedOrderDelivered(ctx context.Context, orderID, placeID string) (string, bool, error)
	InsertOrderTracking(ctx context.Context, p domain.AddTrackingParams) (string, bool, error)
	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	ListOrderTracking(ctx context.Context, orderID string) ([]domain.TrackingEvent, error)
	ReturnStock(ctx context.Context, orderID string, exceptExternalIDs []string) error
}

type EventStream interface {
	Publish(ctx context.Context, ev domain.OrderEvent) error
	Subscribe(ctx context.Context, orderID string) (<-chan domain.OrderEvent, error)
}

type IdempotencyStore interface {
	Acquire(ctx context.Context, key, value string) (stored string, acquired bool, err error)
	Save(ctx context.Context, key, value string) error
	Release(ctx context.Context, key string) error
}
