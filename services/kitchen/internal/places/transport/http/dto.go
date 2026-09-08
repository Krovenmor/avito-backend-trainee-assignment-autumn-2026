package http

type registerPlaceRequest struct {
	Name        string  `json:"name" validate:"required,max=200"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	City        string  `json:"city" validate:"required,max=100"`
	Address     string  `json:"address" validate:"required,max=300"`
	Timezone    string  `json:"timezone" validate:"required,tz"`
	OpensAt     string  `json:"opens_at" validate:"required,datetime=15:04:05"`
	ClosesAt    string  `json:"closes_at" validate:"required,datetime=15:04:05"`
	Type        string  `json:"type" validate:"required,oneof=restaurant cafe shop"`
	WebhookURL  string  `json:"webhook_url" validate:"required,http_url"`
}

type registerPlaceResponse struct {
	PlaceID string `json:"place_id"`
	APIKey  string `json:"api_key"`
}

type updatePlaceRequest struct {
	Name        *string `json:"name" validate:"omitempty,max=200"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	City        *string `json:"city" validate:"omitempty,max=100"`
	Address     *string `json:"address" validate:"omitempty,max=300"`
	Timezone    *string `json:"timezone" validate:"omitempty,tz"`
	OpensAt     *string `json:"opens_at" validate:"omitempty,datetime=15:04:05"`
	ClosesAt    *string `json:"closes_at" validate:"omitempty,datetime=15:04:05"`
	Type        *string `json:"type" validate:"omitempty,oneof=restaurant cafe shop"`
	WebhookURL  *string `json:"webhook_url" validate:"omitempty,http_url"`
	IsActive    *bool   `json:"is_active"`
}

type menuItemRequest struct {
	ExternalID    string  `json:"external_id" validate:"required,max=200"`
	Name          string  `json:"name" validate:"required,max=200"`
	Description   *string `json:"description" validate:"omitempty,max=2000"`
	Price         string  `json:"price" validate:"required,numeric,nonnegative"`
	StockQuantity int32   `json:"stock_quantity" validate:"gte=0"`
}

type menuItemResponse struct {
	ExternalID string `json:"external_id"`
}

type updateMenuItemRequest struct {
	Name        *string `json:"name" validate:"omitempty,max=200"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	Price       *string `json:"price" validate:"omitempty,numeric,nonnegative"`
	StockDelta  *int32  `json:"stock_delta"`
	Reason      string  `json:"reason" validate:"omitempty,max=500"`
}

type placeListItemResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	City        string  `json:"city"`
	Address     string  `json:"address"`
	Timezone    string  `json:"timezone"`
	OpensAt     string  `json:"opens_at"`
	ClosesAt    string  `json:"closes_at"`
	Type        string  `json:"type"`
	IsOpen      bool    `json:"is_open"`
	Rating      *string `json:"rating"`
	RatingCount int32   `json:"rating_count"`
}

type productListItemResponse struct {
	ProductID     string  `json:"product_id"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	Price         string  `json:"price"`
	StockQuantity int32   `json:"stock_quantity"`
}

type productPlaceResponse struct {
	PlaceID string `json:"place_id"`
	Name    string `json:"name"`
	City    string `json:"city"`
	Type    string `json:"type"`
	IsOpen  bool   `json:"is_open"`
}

type productSearchItemResponse struct {
	ProductID     string               `json:"product_id"`
	Name          string               `json:"name"`
	Description   *string              `json:"description"`
	Price         string               `json:"price"`
	StockQuantity int32                `json:"stock_quantity"`
	Place         productPlaceResponse `json:"place"`
}
