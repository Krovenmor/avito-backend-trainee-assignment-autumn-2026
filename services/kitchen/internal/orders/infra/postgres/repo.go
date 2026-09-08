package postgres

import (
	"context"
	"errors"
	"fmt"

	"avito-kitchen/pkg/repo"
	"avito-kitchen/services/kitchen/internal/orders/domain"
	"avito-kitchen/services/kitchen/internal/orders/infra/postgres/queries"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool    *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	queries *queries.Queries
}

func NewRepo(pool *pgxpool.Pool, getter *trmpgx.CtxGetter) (*Repo, error) {
	q, err := queries.GetQueries()
	if err != nil {
		return nil, err
	}
	return &Repo{pool: pool, getter: getter, queries: q}, nil
}

func (r *Repo) conn(ctx context.Context) repo.Querier {
	return r.getter.DefaultTrOrDB(ctx, r.pool)
}

func (r *Repo) UpsertUser(ctx context.Context, userID string) error {
	if _, err := r.conn(ctx).Exec(ctx, r.queries.UpsertUser, userID); err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	return nil
}

func (r *Repo) GetPlaceForOrder(ctx context.Context, placeID string) (*domain.PlaceForOrder, error) {
	row, err := repo.GetQueryByName[placeForOrderRow](ctx, r.conn(ctx), r.queries.GetPlaceForOrder, placeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get place for order: %w", err)
	}

	return toPlaceForOrder(*row), nil
}

func (r *Repo) LockPlaceProducts(ctx context.Context, placeID string, productIDs []string) ([]string, error) {
	found, err := repo.GetSliceQueryByType[string](ctx, r.conn(ctx), r.queries.LockOrderProducts, placeID, productIDs)
	if err != nil {
		return nil, fmt.Errorf("lock place products: %w", err)
	}
	return found, nil
}

func (r *Repo) ReserveStock(ctx context.Context, items []domain.CreateOrderItemInput) ([]domain.UnavailableItem, error) {
	cols := toReserveColumns(items)

	rows, err := repo.GetSliceQueryByName[unavailableItemRow](ctx, r.conn(ctx), r.queries.ReserveStock,
		cols.productIDs, cols.quantities,
	)
	if err != nil {
		return nil, fmt.Errorf("reserve stock: %w", err)
	}

	return toUnavailableItems(rows), nil
}

func (r *Repo) InsertOrder(ctx context.Context, p domain.CreateOrderParams) (*domain.OrderTotal, error) {
	cols := toOrderItemColumns(p.Items)

	row, err := repo.GetQueryByName[orderTotalRow](ctx, r.conn(ctx), r.queries.InsertOrder,
		p.OrderID, p.UserID, p.PlaceID, cols.ids, cols.productIDs, cols.quantities, p.LockedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	return toOrderTotal(*row), nil
}

func (r *Repo) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	row, err := repo.GetQueryByName[orderRow](ctx, r.conn(ctx), r.queries.GetOrder, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	items, err := repo.GetSliceQueryByName[orderItemRow](ctx, r.conn(ctx), r.queries.GetOrderItems, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}

	return toOrder(*row, items), nil
}

func (r *Repo) GetOrderOwner(ctx context.Context, orderID string) (*domain.OrderOwner, error) {
	row, err := repo.GetQueryByName[orderOwnerRow](ctx, r.conn(ctx), r.queries.GetOrderOwnerStatus, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order owner: %w", err)
	}

	return toOrderOwner(*row), nil
}

func (r *Repo) GetOrderStatusForPlace(ctx context.Context, orderID, placeID string) (domain.OrderStatus, error) {
	var status string
	err := r.conn(ctx).QueryRow(ctx, r.queries.GetOrderStatus, orderID, placeID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get order status for place: %w", err)
	}

	return domain.OrderStatus(status), nil
}

func (r *Repo) ListUserOrders(ctx context.Context, p domain.ListUserOrdersParams) ([]domain.OrderSummary, error) {
	rows, err := repo.GetSliceQueryByName[orderSummaryRow](ctx, r.conn(ctx), r.queries.ListUserOrders, p.UserID, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("list user orders: %w", err)
	}

	return toOrderSummaries(rows), nil
}

func (r *Repo) ClaimPendingOrders(ctx context.Context, p domain.ClaimPendingOrdersParams) ([]domain.PendingOrder, error) {
	rows, err := repo.GetSliceQueryByName[pendingOrderRow](ctx, r.conn(ctx), r.queries.ListPendingOrders,
		p.Owner, p.LeaseTTL.Seconds(), p.Limit, p.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("claim pending orders: %w", err)
	}

	return toPendingOrders(rows), nil
}

func (r *Repo) RenewOrderLeases(ctx context.Context, owner string, orderIDs []string) error {
	if len(orderIDs) == 0 {
		return nil
	}

	if _, err := r.conn(ctx).Exec(ctx, r.queries.RenewOrderLeases, orderIDs, owner); err != nil {
		return fmt.Errorf("renew order leases: %w", err)
	}
	return nil
}

func (r *Repo) CancelPendingOrder(ctx context.Context, orderID, userID string) (bool, error) {
	tag, err := r.conn(ctx).Exec(ctx, r.queries.CancelOrder, orderID, userID)
	if err != nil {
		return false, fmt.Errorf("cancel pending order: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) UpdatePendingOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) (bool, error) {
	tag, err := r.conn(ctx).Exec(ctx, r.queries.UpdateOrderStatus, orderID, string(status))
	if err != nil {
		return false, fmt.Errorf("update pending order status: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) MarkAcceptedOrderDelivered(ctx context.Context, orderID, placeID string) (string, bool, error) {
	var userID string
	err := r.conn(ctx).QueryRow(ctx, r.queries.MarkDelivered, orderID, placeID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("mark accepted order delivered: %w", err)
	}

	return userID, true, nil
}

func (r *Repo) ReturnStock(ctx context.Context, orderID string, exceptExternalIDs []string) error {
	if _, err := r.conn(ctx).Exec(ctx, r.queries.ReturnStock, orderID, toTextArray(exceptExternalIDs)); err != nil {
		return fmt.Errorf("return stock: %w", err)
	}
	return nil
}

func (r *Repo) ZeroStock(ctx context.Context, orderID string, externalIDs []string) ([]string, error) {
	zeroed, err := repo.GetSliceQueryByType[string](ctx, r.conn(ctx), r.queries.ZeroStock, orderID, toTextArray(externalIDs))
	if err != nil {
		return nil, fmt.Errorf("zero stock: %w", err)
	}
	return zeroed, nil
}

func (r *Repo) InsertOrderRating(ctx context.Context, p domain.RateOrderParams) (string, bool, error) {
	var placeID string
	err := r.conn(ctx).QueryRow(ctx, r.queries.InsertOrderRating,
		p.RatingID, p.OrderID, p.UserID, p.Score,
	).Scan(&placeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("insert order rating: %w", err)
	}

	return placeID, true, nil
}

func (r *Repo) ListOrderTracking(ctx context.Context, orderID string) ([]domain.TrackingEvent, error) {
	rows, err := repo.GetSliceQueryByName[trackingEventRow](ctx, r.conn(ctx), r.queries.ListOrderTracking, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order tracking: %w", err)
	}

	return toTrackingEvents(rows), nil
}

func (r *Repo) InsertOrderTracking(ctx context.Context, p domain.AddTrackingParams) (string, bool, error) {
	var userID string
	err := r.conn(ctx).QueryRow(ctx, r.queries.InsertOrderTracking,
		p.TrackingID, p.OrderID, p.PlaceID, string(p.Source), p.ExternalCode, string(p.Status), p.OccurredAt,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("insert order tracking: %w", err)
	}

	return userID, true, nil
}

func (r *Repo) AddPlaceRating(ctx context.Context, placeID string, score int16) error {
	if _, err := r.conn(ctx).Exec(ctx, r.queries.AddPlaceRating, placeID, score); err != nil {
		return fmt.Errorf("add place rating: %w", err)
	}
	return nil
}
