package domain

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusAccepted  OrderStatus = "accepted"
	StatusRejected  OrderStatus = "rejected"
	StatusCompleted OrderStatus = "completed"
	StatusCancelled OrderStatus = "cancelled"
	StatusExpired   OrderStatus = "expired"
)

const ReasonOutOfStock = "out_of_stock"

type OrderItem struct {
	ProductID  string
	ExternalID string
	Name       string
	Quantity   int32
	Price      string
}

type Order struct {
	ID          string
	UserID      string
	PlaceID     string
	PlaceName   string
	Status      OrderStatus
	TotalAmount string
	CreatedAt   time.Time
	Items       []OrderItem
}

type OrderSummary struct {
	ID          string
	PlaceID     string
	PlaceName   string
	Status      OrderStatus
	TotalAmount string
	CreatedAt   time.Time
}

type PendingOrder struct {
	ID         string
	UserID     string
	PlaceID    string
	CreatedAt  time.Time
	WebhookURL string
}

type EventSource string

const (
	SourcePlatform EventSource = "platform"
	SourcePlace    EventSource = "place"
	SourceDelivery EventSource = "delivery"
)

type OrderEvent struct {
	OrderID string
	UserID  string
	Source  EventSource
	Status  string
}

type TrackingStatus string

type AddTrackingInput struct {
	OrderID      string
	PlaceID      string
	ExternalCode string
	Status       TrackingStatus
	OccurredAt   time.Time
}

type AddTrackingParams struct {
	TrackingID   string
	OrderID      string
	PlaceID      string
	Source       EventSource
	ExternalCode string
	Status       TrackingStatus
	OccurredAt   time.Time
}

type Decision struct {
	Status                 OrderStatus
	Reason                 string
	UnavailableExternalIDs []string
}

type CreateOrderItemInput struct {
	ProductID string
	Quantity  int32
}

type CreateOrderInput struct {
	IdempotencyKey string
	BodyHash       string
	UserID         string
	PlaceID        string
	Items          []CreateOrderItemInput
}

type CreateOrderOutput struct {
	OrderID string
	Status  OrderStatus
}

type CreateOrderParams struct {
	OrderID  string
	UserID   string
	PlaceID  string
	LockedBy string
	Items    []CreateOrderItemInput
}

type ClaimPendingOrdersParams struct {
	Owner    string
	LeaseTTL time.Duration
	Limit    int32
	Active   []string
}

type PlaceForOrder struct {
	WebhookURL string
	IsActive   bool
	IsOpen     bool
}

type OrderTotal struct {
	TotalAmount string
	CreatedAt   time.Time
}

type OrderOwner struct {
	UserID string
	Status OrderStatus
}

type ListUserOrdersParams struct {
	UserID string
	Limit  int32
	Offset int32
}

type UnavailableItem struct {
	ProductID string
	Requested int32
	Available int32
}

type RateOrderInput struct {
	OrderID string
	UserID  string
	Score   int16
}

type RateOrderParams struct {
	RatingID string
	OrderID  string
	UserID   string
	Score    int16
}

type InstanceID string

type TrackingEvent struct {
	Source       EventSource
	ExternalCode string
	Status       TrackingStatus
	OccurredAt   time.Time
}
