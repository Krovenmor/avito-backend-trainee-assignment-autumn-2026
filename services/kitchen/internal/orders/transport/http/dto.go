package http

import "time"

type createOrderItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid4_rfc4122"`
	Quantity  int32  `json:"quantity" validate:"required,gt=0,lte=1000"`
}

type createOrderRequest struct {
	UserID  string                   `json:"user_id" validate:"required,uuid4_rfc4122"`
	PlaceID string                   `json:"place_id" validate:"required,uuid4_rfc4122"`
	Items   []createOrderItemRequest `json:"items" validate:"required,min=1,max=100,dive"`
}

type createOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type cancelOrderRequest struct {
	UserID string `json:"user_id" validate:"required,uuid4_rfc4122"`
}

type orderListItemResponse struct {
	OrderID     string    `json:"order_id"`
	PlaceID     string    `json:"place_id"`
	PlaceName   string    `json:"place_name"`
	Status      string    `json:"status"`
	TotalAmount string    `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

type orderEventResponse struct {
	OrderID string `json:"order_id"`
	Source  string `json:"source"`
	Status  string `json:"status"`
}

type addTrackingRequest struct {
	ExternalCode string    `json:"external_code" validate:"required,max=64"`
	Status       string    `json:"status" validate:"required,oneof=accepted cooking ready handed_over"`
	OccurredAt   time.Time `json:"occurred_at" validate:"required"`
}

type unavailableItemResponse struct {
	ProductID string `json:"product_id"`
	Requested int32  `json:"requested"`
	Available int32  `json:"available"`
}

type outOfStockResponse struct {
	Error string                    `json:"error"`
	Items []unavailableItemResponse `json:"items"`
}

type rateOrderRequest struct {
	UserID string `json:"user_id" validate:"required,uuid4_rfc4122"`
	Score  int16  `json:"score" validate:"required,min=1,max=5"`
}

type orderItemResponse struct {
	ProductID  string `json:"product_id"`
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Quantity   int32  `json:"quantity"`
	Price      string `json:"price"`
}

type orderResponse struct {
	OrderID     string              `json:"order_id"`
	PlaceID     string              `json:"place_id"`
	PlaceName   string              `json:"place_name"`
	Status      string              `json:"status"`
	TotalAmount string              `json:"total_amount"`
	CreatedAt   time.Time           `json:"created_at"`
	Items       []orderItemResponse `json:"items"`
}

type trackingEventResponse struct {
	Source       string    `json:"source"`
	ExternalCode string    `json:"external_code"`
	Status       string    `json:"status"`
	OccurredAt   time.Time `json:"occurred_at"`
}
