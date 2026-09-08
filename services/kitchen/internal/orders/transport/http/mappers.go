package http

import (
	"avito-kitchen/pkg/slice"
	"avito-kitchen/services/kitchen/internal/httpx"
	"avito-kitchen/services/kitchen/internal/orders/domain"
)

func toCreateOrderInput(idempotencyKey, bodyHash string, req createOrderRequest) domain.CreateOrderInput {
	return domain.CreateOrderInput{
		IdempotencyKey: httpx.NormalizeUUID(idempotencyKey),
		BodyHash:       bodyHash,
		UserID:         httpx.NormalizeUUID(req.UserID),
		PlaceID:        httpx.NormalizeUUID(req.PlaceID),
		Items:          slice.Map(req.Items, toCreateOrderItemInput),
	}
}

func toCreateOrderItemInput(item createOrderItemRequest) domain.CreateOrderItemInput {
	return domain.CreateOrderItemInput{
		ProductID: httpx.NormalizeUUID(item.ProductID),
		Quantity:  item.Quantity,
	}
}

func toCreateOrderResponse(out *domain.CreateOrderOutput) createOrderResponse {
	return createOrderResponse{OrderID: out.OrderID, Status: string(out.Status)}
}

func toListUserOrdersParams(userID string, limit, offset int32) domain.ListUserOrdersParams {
	return domain.ListUserOrdersParams{UserID: userID, Limit: limit, Offset: offset}
}

func toOrderListItemResponse(order domain.OrderSummary) orderListItemResponse {
	return orderListItemResponse{
		OrderID:     order.ID,
		PlaceID:     order.PlaceID,
		PlaceName:   order.PlaceName,
		Status:      string(order.Status),
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt,
	}
}

func toOrderListItemResponses(orders []domain.OrderSummary) []orderListItemResponse {
	return slice.Map(orders, toOrderListItemResponse)
}

func toUnavailableItemResponse(item domain.UnavailableItem) unavailableItemResponse {
	return unavailableItemResponse{
		ProductID: item.ProductID,
		Requested: item.Requested,
		Available: item.Available,
	}
}

func toOutOfStockResponse(items []domain.UnavailableItem) outOfStockResponse {
	return outOfStockResponse{
		Error: outOfStockMessage,
		Items: slice.Map(items, toUnavailableItemResponse),
	}
}

func toOrderEventResponse(ev domain.OrderEvent) orderEventResponse {
	return orderEventResponse{OrderID: ev.OrderID, Source: string(ev.Source), Status: ev.Status}
}

func toAddTrackingInput(orderID, placeID string, req addTrackingRequest) domain.AddTrackingInput {
	return domain.AddTrackingInput{
		OrderID:      orderID,
		PlaceID:      placeID,
		ExternalCode: req.ExternalCode,
		Status:       domain.TrackingStatus(req.Status),
		OccurredAt:   req.OccurredAt,
	}
}

func toOrderItemResponse(item domain.OrderItem) orderItemResponse {
	return orderItemResponse{
		ProductID:  item.ProductID,
		ExternalID: item.ExternalID,
		Name:       item.Name,
		Quantity:   item.Quantity,
		Price:      item.Price,
	}
}

func toOrderResponse(order *domain.Order) orderResponse {
	return orderResponse{
		OrderID:     order.ID,
		PlaceID:     order.PlaceID,
		PlaceName:   order.PlaceName,
		Status:      string(order.Status),
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt,
		Items:       slice.Map(order.Items, toOrderItemResponse),
	}
}

func toTrackingEventResponse(event domain.TrackingEvent) trackingEventResponse {
	return trackingEventResponse{
		Source:       string(event.Source),
		ExternalCode: event.ExternalCode,
		Status:       string(event.Status),
		OccurredAt:   event.OccurredAt,
	}
}

func toTrackingEventResponses(events []domain.TrackingEvent) []trackingEventResponse {
	return slice.Map(events, toTrackingEventResponse)
}
