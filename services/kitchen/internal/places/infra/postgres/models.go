package postgres

type placeAuthRow struct {
	ID       string `db:"id"`
	IsActive bool   `db:"is_active"`
}

type placeSummaryRow struct {
	ID          string  `db:"id"`
	Name        string  `db:"name"`
	Description *string `db:"description"`
	City        string  `db:"city"`
	Address     string  `db:"address"`
	Timezone    string  `db:"timezone"`
	OpensAt     string  `db:"opens_at"`
	ClosesAt    string  `db:"closes_at"`
	Type        string  `db:"place_type"`
	IsOpen      bool    `db:"is_open"`
	Rating      *string `db:"rating"`
	RatingCount int32   `db:"rating_count"`
}

type menuItemRow struct {
	ProductID     string  `db:"product_id"`
	Name          string  `db:"name"`
	Description   *string `db:"description"`
	Price         string  `db:"price"`
	StockQuantity int32   `db:"stock_quantity"`
}

type productSearchRow struct {
	ProductID     string  `db:"product_id"`
	Name          string  `db:"name"`
	Description   *string `db:"description"`
	Price         string  `db:"price"`
	StockQuantity int32   `db:"stock_quantity"`
	PlaceID       string  `db:"place_id"`
	PlaceName     string  `db:"place_name"`
	PlaceCity     string  `db:"place_city"`
	PlaceType     string  `db:"place_type"`
	PlaceIsOpen   bool    `db:"place_is_open"`
}

type insertProductsColumns struct {
	ids             []string
	externalIDs     []string
	names           []string
	descriptions    []*string
	prices          []string
	stockQuantities []int32
}
