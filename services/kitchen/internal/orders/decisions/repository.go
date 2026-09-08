package decisions

import (
	"context"

	"avito-kitchen/services/kitchen/internal/orders/domain"
)

type Repository interface {
	ClaimPendingOrders(ctx context.Context, p domain.ClaimPendingOrdersParams) ([]domain.PendingOrder, error)
	RenewOrderLeases(ctx context.Context, owner string, orderIDs []string) error

	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	UpdatePendingOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) (bool, error)

	ReturnStock(ctx context.Context, orderID string, exceptExternalIDs []string) error
	ZeroStock(ctx context.Context, orderID string, externalIDs []string) ([]string, error)
}
