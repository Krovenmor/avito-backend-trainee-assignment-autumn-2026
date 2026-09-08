package postgres

import (
	"avito-kitchen/pkg/slice"
	"avito-kitchen/services/kitchen/internal/orders/domain"

	"github.com/google/uuid"
)

func toUnavailableItem(row unavailableItemRow) domain.UnavailableItem {
	return domain.UnavailableItem{
		ProductID: row.ProductID,
		Requested: row.Requested,
		Available: row.Available,
	}
}

func toUnavailableItems(rows []unavailableItemRow) []domain.UnavailableItem {
	return slice.Map(rows, toUnavailableItem)
}

func toReserveColumns(items []domain.CreateOrderItemInput) reserveColumns {
	cols := reserveColumns{
		productIDs: make([]string, len(items)),
		quantities: make([]int32, len(items)),
	}
	for i, item := range items {
		cols.productIDs[i] = item.ProductID
		cols.quantities[i] = item.Quantity
	}
	return cols
}

func toOrderItemColumns(items []domain.CreateOrderItemInput) orderItemColumns {
	cols := orderItemColumns{
		ids:        make([]string, len(items)),
		productIDs: make([]string, len(items)),
		quantities: make([]int32, len(items)),
	}
	for i, item := range items {
		cols.ids[i] = uuid.NewString()
		cols.productIDs[i] = item.ProductID
		cols.quantities[i] = item.Quantity
	}
	return cols
}

func toPlaceForOrder(row placeForOrderRow) *domain.PlaceForOrder {
	return &domain.PlaceForOrder{
		WebhookURL: row.WebhookURL,
		IsActive:   row.IsActive,
		IsOpen:     row.IsOpen,
	}
}

func toOrderTotal(row orderTotalRow) *domain.OrderTotal {
	return &domain.OrderTotal{TotalAmount: row.TotalAmount, CreatedAt: row.CreatedAt}
}

func toTextArray(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func toOrderOwner(row orderOwnerRow) *domain.OrderOwner {
	return &domain.OrderOwner{UserID: row.UserID, Status: domain.OrderStatus(row.Status)}
}

func toOrder(row orderRow, items []orderItemRow) *domain.Order {
	return &domain.Order{
		ID:          row.ID,
		UserID:      row.UserID,
		PlaceID:     row.PlaceID,
		PlaceName:   row.PlaceName,
		Status:      domain.OrderStatus(row.Status),
		TotalAmount: row.TotalAmount,
		CreatedAt:   row.CreatedAt,
		Items:       toOrderItems(items),
	}
}

func toOrderItem(row orderItemRow) domain.OrderItem {
	return domain.OrderItem{
		ProductID:  row.ProductID,
		ExternalID: row.ExternalID,
		Name:       row.Name,
		Quantity:   row.Quantity,
		Price:      row.Price,
	}
}

func toOrderItems(rows []orderItemRow) []domain.OrderItem {
	return slice.Map(rows, toOrderItem)
}

func toOrderSummary(row orderSummaryRow) domain.OrderSummary {
	return domain.OrderSummary{
		ID:          row.ID,
		PlaceID:     row.PlaceID,
		PlaceName:   row.PlaceName,
		Status:      domain.OrderStatus(row.Status),
		TotalAmount: row.TotalAmount,
		CreatedAt:   row.CreatedAt,
	}
}

func toOrderSummaries(rows []orderSummaryRow) []domain.OrderSummary {
	return slice.Map(rows, toOrderSummary)
}

func toPendingOrder(row pendingOrderRow) domain.PendingOrder {
	return domain.PendingOrder{
		ID:         row.ID,
		UserID:     row.UserID,
		PlaceID:    row.PlaceID,
		CreatedAt:  row.CreatedAt,
		WebhookURL: row.WebhookURL,
	}
}

func toPendingOrders(rows []pendingOrderRow) []domain.PendingOrder {
	return slice.Map(rows, toPendingOrder)
}

func toTrackingEvent(row trackingEventRow) domain.TrackingEvent {
	return domain.TrackingEvent{
		Source:       domain.EventSource(row.Source),
		ExternalCode: row.ExternalCode,
		Status:       domain.TrackingStatus(row.Status),
		OccurredAt:   row.OccurredAt,
	}
}

func toTrackingEvents(rows []trackingEventRow) []domain.TrackingEvent {
	return slice.Map(rows, toTrackingEvent)
}
