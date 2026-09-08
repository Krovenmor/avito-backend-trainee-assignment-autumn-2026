package postgres

import (
	"context"
	"errors"
	"fmt"

	"avito-kitchen/pkg/repo"
	"avito-kitchen/services/kitchen/internal/places/domain"
	"avito-kitchen/services/kitchen/internal/places/infra/postgres/queries"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	uniqueViolation = "23505"
	checkViolation  = "23514"

	priceCheckConstraint = "products_price_non_negative"
	stockCheckConstraint = "products_stock_non_negative"
)

type Repo struct {
	pool    *pgxpool.Pool
	queries *queries.Queries
}

func NewRepo(pool *pgxpool.Pool) (*Repo, error) {
	q, err := queries.GetQueries()
	if err != nil {
		return nil, err
	}
	return &Repo{pool: pool, queries: q}, nil
}

func (r *Repo) InsertPlace(ctx context.Context, p domain.Place) error {
	_, err := r.pool.Exec(ctx, r.queries.InsertPlace,
		p.ID, p.Name, p.Description, p.City, p.Address, p.Timezone,
		p.OpensAt, p.ClosesAt, p.Type, p.WebhookURL, p.APIKeyHash,
	)
	if err != nil {
		return fmt.Errorf("insert place: %w", err)
	}
	return nil
}

func (r *Repo) GetPlaceByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.PlaceAuth, error) {
	row, err := repo.GetQueryByName[placeAuthRow](ctx, r.pool, r.queries.GetPlaceByAPIKeyHash, apiKeyHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get place by api key hash: %w", err)
	}

	return toPlaceAuth(*row), nil
}

func (r *Repo) UpdatePlace(ctx context.Context, p domain.UpdatePlaceParams) error {
	tag, err := r.pool.Exec(ctx, r.queries.UpdatePlace,
		p.ID, p.Name, p.Description, p.City, p.Address, p.Timezone,
		p.OpensAt, p.ClosesAt, p.Type, p.WebhookURL, p.IsActive,
	)
	if err != nil {
		return fmt.Errorf("update place: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repo) GetPlace(ctx context.Context, placeID string) (*domain.PlaceSummary, error) {
	row, err := repo.GetQueryByName[placeSummaryRow](ctx, r.pool, r.queries.GetPlace, placeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get place: %w", err)
	}

	place := toPlaceSummary(*row)
	return &place, nil
}

func (r *Repo) ListPlaces(ctx context.Context, p domain.ListPlacesParams) ([]domain.PlaceSummary, error) {
	rows, err := repo.GetSliceQueryByName[placeSummaryRow](ctx, r.pool, r.queries.ListPlaces,
		p.City, p.Type, p.Search, p.MinRating, p.SortByRating, p.Limit, p.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list places: %w", err)
	}

	return toPlaceSummaries(rows), nil
}

func (r *Repo) PlaceExists(ctx context.Context, placeID string) (bool, error) {
	var id string
	err := r.pool.QueryRow(ctx, r.queries.PlaceExists, placeID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("place exists: %w", err)
	}
	return true, nil
}

func (r *Repo) InsertProducts(ctx context.Context, placeID string, products []domain.InsertProductParams) error {
	cols := toInsertProductsColumns(products)

	_, err := r.pool.Exec(ctx, r.queries.InsertProducts,
		placeID, cols.ids, cols.externalIDs, cols.names, cols.descriptions, cols.prices, cols.stockQuantities,
	)
	if err != nil {
		if svcErr := toServiceErr(err); svcErr != nil {
			return svcErr
		}
		return fmt.Errorf("insert products: %w", err)
	}
	return nil
}

func (r *Repo) UpdateProductMenu(ctx context.Context, p domain.UpdateProductMenuParams) error {
	tag, err := r.pool.Exec(ctx, r.queries.UpdateProductMenu,
		p.ExternalID, p.PlaceID, p.Name, p.Description, p.Price, p.StockDelta,
	)
	if err != nil {
		if svcErr := toServiceErr(err); svcErr != nil {
			return svcErr
		}
		return fmt.Errorf("update product menu: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repo) SoftDeleteProduct(ctx context.Context, externalID, placeID string) error {
	tag, err := r.pool.Exec(ctx, r.queries.SoftDeleteProduct, externalID, placeID)
	if err != nil {
		return fmt.Errorf("soft delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repo) ListProducts(ctx context.Context, p domain.ListProductsParams) ([]domain.MenuItem, error) {
	rows, err := repo.GetSliceQueryByName[menuItemRow](ctx, r.pool, r.queries.ListProducts,
		p.PlaceID, p.Search, p.Limit, p.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	return toMenuItems(rows), nil
}

func (r *Repo) SearchProducts(ctx context.Context, p domain.SearchProductsParams) ([]domain.ProductSearchItem, error) {
	rows, err := repo.GetSliceQueryByName[productSearchRow](ctx, r.pool, r.queries.SearchProducts,
		p.City, p.Type, p.Search, p.Limit, p.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}

	return toProductSearchItems(rows), nil
}
