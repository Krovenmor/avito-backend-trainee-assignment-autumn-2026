package client

import (
	"time"

	"avito-kitchen/services/testapi/internal/stock"
)

type RegisterPlaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	City        string `json:"city"`
	Address     string `json:"address"`
	Timezone    string `json:"timezone"`
	OpensAt     string `json:"opens_at"`
	ClosesAt    string `json:"closes_at"`
	Type        string `json:"type"`
	WebhookURL  string `json:"webhook_url"`
}

type RegisterPlaceResponse struct {
	PlaceID string `json:"place_id"`
	APIKey  string `json:"api_key"`
}

type menuItemRequest struct {
	ExternalID    string `json:"external_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         string `json:"price"`
	StockQuantity int32  `json:"stock_quantity"`
}

func toMenuRequest(items []stock.Item) []menuItemRequest {
	menu := make([]menuItemRequest, len(items))
	for i, item := range items {
		menu[i] = menuItemRequest{
			ExternalID:    item.ExternalID,
			Name:          item.Name,
			Description:   item.Description,
			Price:         item.Price,
			StockQuantity: item.Quantity,
		}
	}
	return menu
}

type TrackingRequest struct {
	ExternalCode string    `json:"external_code"`
	Status       string    `json:"status"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type StockDeltaRequest struct {
	StockDelta int32 `json:"stock_delta"`
}
