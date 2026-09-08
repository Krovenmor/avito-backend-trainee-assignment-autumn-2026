package webhook

import (
	"avito-kitchen/pkg/slice"
	"avito-kitchen/services/kitchen/internal/orders/domain"
)

type itemPayload struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Quantity   int32  `json:"quantity"`
	Price      string `json:"price"`
}

type orderPayload struct {
	OrderID     string        `json:"order_id"`
	PlaceID     string        `json:"place_id"`
	TotalAmount string        `json:"total_amount"`
	Items       []itemPayload `json:"items"`
}

type decisionPayload struct {
	Decision               string   `json:"decision"`
	Reason                 string   `json:"reason"`
	UnavailableExternalIDs []string `json:"unavailable_external_ids"`
}

func toItemPayload(item domain.OrderItem) itemPayload {
	return itemPayload{
		ExternalID: item.ExternalID,
		Name:       item.Name,
		Quantity:   item.Quantity,
		Price:      item.Price,
	}
}

func toOrderPayload(order domain.Order) orderPayload {
	return orderPayload{
		OrderID:     order.ID,
		PlaceID:     order.PlaceID,
		TotalAmount: order.TotalAmount,
		Items:       slice.Map(order.Items, toItemPayload),
	}
}

func toDecision(payload decisionPayload) (domain.Decision, bool) {
	switch payload.Decision {
	case decisionAccepted:
		return domain.Decision{Status: domain.StatusAccepted}, true
	case decisionRejected:
		return domain.Decision{
			Status:                 domain.StatusRejected,
			Reason:                 payload.Reason,
			UnavailableExternalIDs: payload.UnavailableExternalIDs,
		}, true
	default:
		return domain.Decision{}, false
	}
}
