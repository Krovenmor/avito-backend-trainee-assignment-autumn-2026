package domain

type Place struct {
	ID          string
	Name        string
	Description *string
	City        string
	Address     string
	Timezone    string
	OpensAt     string
	ClosesAt    string
	Type        string
	WebhookURL  string
	APIKeyHash  string
}

type PlaceAuth struct {
	ID       string
	IsActive bool
}

type UpdatePlaceParams struct {
	ID          string
	Name        *string
	Description *string
	City        *string
	Address     *string
	Timezone    *string
	OpensAt     *string
	ClosesAt    *string
	Type        *string
	WebhookURL  *string
	IsActive    *bool
}

type InsertProductParams struct {
	ID            string
	ExternalID    string
	Name          string
	Description   *string
	Price         string
	StockQuantity int32
}

type UpdateProductMenuParams struct {
	ExternalID  string
	PlaceID     string
	Name        *string
	Description *string
	Price       *string
	StockDelta  *int32
}

type RegisterPlaceInput struct {
	Name        string
	Description *string
	City        string
	Address     string
	Timezone    string
	OpensAt     string
	ClosesAt    string
	Type        string
	WebhookURL  string
}

type RegisterPlaceOutput struct {
	PlaceID string
	APIKey  string
}

type UpdatePlaceInput struct {
	PlaceID     string
	Name        *string
	Description *string
	City        *string
	Address     *string
	Timezone    *string
	OpensAt     *string
	ClosesAt    *string
	Type        *string
	WebhookURL  *string
	IsActive    *bool
}

type MenuItemInput struct {
	ExternalID    string
	Name          string
	Description   *string
	Price         string
	StockQuantity int32
}

type UpdateMenuItemInput struct {
	PlaceID     string
	ExternalID  string
	Name        *string
	Description *string
	Price       *string
	StockDelta  *int32
	Reason      string
}

type PlaceSummary struct {
	ID          string
	Name        string
	Description *string
	City        string
	Address     string
	Timezone    string
	OpensAt     string
	ClosesAt    string
	Type        string
	IsOpen      bool
	Rating      *string
	RatingCount int32
}

type ListPlacesParams struct {
	City         *string
	Type         *string
	Search       *string
	MinRating    *string
	SortByRating bool
	Limit        int32
	Offset       int32
}

type MenuItem struct {
	ProductID     string
	Name          string
	Description   *string
	Price         string
	StockQuantity int32
}

type ListProductsParams struct {
	PlaceID string
	Search  *string
	Limit   int32
	Offset  int32
}

type ProductPlace struct {
	ID     string
	Name   string
	City   string
	Type   string
	IsOpen bool
}

type ProductSearchItem struct {
	ProductID     string
	Name          string
	Description   *string
	Price         string
	StockQuantity int32
	Place         ProductPlace
}

type SearchProductsParams struct {
	City   *string
	Type   *string
	Search *string
	Limit  int32
	Offset int32
}
