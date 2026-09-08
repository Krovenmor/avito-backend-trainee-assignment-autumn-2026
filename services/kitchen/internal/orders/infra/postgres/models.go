package postgres

import "time"

type orderRow struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	PlaceID     string    `db:"place_id"`
	PlaceName   string    `db:"place_name"`
	Status      string    `db:"order_status"`
	TotalAmount string    `db:"total_amount"`
	CreatedAt   time.Time `db:"created_at"`
}

type orderItemRow struct {
	ProductID  string `db:"product_id"`
	ExternalID string `db:"external_id"`
	Name       string `db:"name"`
	Quantity   int32  `db:"quantity"`
	Price      string `db:"price"`
}

type orderSummaryRow struct {
	ID          string    `db:"id"`
	PlaceID     string    `db:"place_id"`
	PlaceName   string    `db:"place_name"`
	Status      string    `db:"order_status"`
	TotalAmount string    `db:"total_amount"`
	CreatedAt   time.Time `db:"created_at"`
}

type pendingOrderRow struct {
	ID         string    `db:"id"`
	UserID     string    `db:"user_id"`
	PlaceID    string    `db:"place_id"`
	CreatedAt  time.Time `db:"created_at"`
	WebhookURL string    `db:"webhook_url"`
}

type orderTotalRow struct {
	TotalAmount string    `db:"total_amount"`
	CreatedAt   time.Time `db:"created_at"`
}

type placeForOrderRow struct {
	WebhookURL string `db:"webhook_url"`
	IsActive   bool   `db:"is_active"`
	IsOpen     bool   `db:"is_open"`
}

type orderOwnerRow struct {
	UserID string `db:"user_id"`
	Status string `db:"order_status"`
}

type unavailableItemRow struct {
	ProductID string `db:"product_id"`
	Requested int32  `db:"requested"`
	Available int32  `db:"available"`
}

type reserveColumns struct {
	productIDs []string
	quantities []int32
}

type orderItemColumns struct {
	ids        []string
	productIDs []string
	quantities []int32
}

type trackingEventRow struct {
	Source       string    `db:"source"`
	ExternalCode string    `db:"external_code"`
	Status       string    `db:"status"`
	OccurredAt   time.Time `db:"occurred_at"`
}
