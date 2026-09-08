package postgres

import (
	"errors"

	"avito-kitchen/pkg/slice"
	"avito-kitchen/services/kitchen/internal/places/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

func toProductSearchItem(row productSearchRow) domain.ProductSearchItem {
	return domain.ProductSearchItem{
		ProductID:     row.ProductID,
		Name:          row.Name,
		Description:   row.Description,
		Price:         row.Price,
		StockQuantity: row.StockQuantity,
		Place: domain.ProductPlace{
			ID:     row.PlaceID,
			Name:   row.PlaceName,
			City:   row.PlaceCity,
			Type:   row.PlaceType,
			IsOpen: row.PlaceIsOpen,
		},
	}
}

func toProductSearchItems(rows []productSearchRow) []domain.ProductSearchItem {
	return slice.Map(rows, toProductSearchItem)
}

func toPlaceAuth(row placeAuthRow) *domain.PlaceAuth {
	return &domain.PlaceAuth{ID: row.ID, IsActive: row.IsActive}
}

func toPlaceSummary(row placeSummaryRow) domain.PlaceSummary {
	return domain.PlaceSummary{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		City:        row.City,
		Address:     row.Address,
		Timezone:    row.Timezone,
		OpensAt:     row.OpensAt,
		ClosesAt:    row.ClosesAt,
		Type:        row.Type,
		IsOpen:      row.IsOpen,
		Rating:      row.Rating,
		RatingCount: row.RatingCount,
	}
}

func toPlaceSummaries(rows []placeSummaryRow) []domain.PlaceSummary {
	return slice.Map(rows, toPlaceSummary)
}

func toMenuItem(row menuItemRow) domain.MenuItem {
	return domain.MenuItem{
		ProductID:     row.ProductID,
		Name:          row.Name,
		Description:   row.Description,
		Price:         row.Price,
		StockQuantity: row.StockQuantity,
	}
}

func toMenuItems(rows []menuItemRow) []domain.MenuItem {
	return slice.Map(rows, toMenuItem)
}

func toServiceErr(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}

	switch {
	case pgErr.Code == uniqueViolation:
		return domain.ErrDuplicateExternalID
	case pgErr.Code == checkViolation && pgErr.ConstraintName == priceCheckConstraint:
		return domain.ErrInvalidPrice
	case pgErr.Code == checkViolation && pgErr.ConstraintName == stockCheckConstraint:
		return domain.ErrStockConflict
	default:
		return nil
	}
}

func toInsertProductsColumns(products []domain.InsertProductParams) insertProductsColumns {
	cols := insertProductsColumns{
		ids:             make([]string, len(products)),
		externalIDs:     make([]string, len(products)),
		names:           make([]string, len(products)),
		descriptions:    make([]*string, len(products)),
		prices:          make([]string, len(products)),
		stockQuantities: make([]int32, len(products)),
	}
	for i, p := range products {
		cols.ids[i] = p.ID
		cols.externalIDs[i] = p.ExternalID
		cols.names[i] = p.Name
		cols.descriptions[i] = p.Description
		cols.prices[i] = p.Price
		cols.stockQuantities[i] = p.StockQuantity
	}
	return cols
}
