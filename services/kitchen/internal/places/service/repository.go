package service

import (
	"avito-kitchen/services/kitchen/internal/places/domain"
	"context"
)

type Repository interface {
	InsertPlace(ctx context.Context, p domain.Place) error
	GetPlaceByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.PlaceAuth, error)
	UpdatePlace(ctx context.Context, p domain.UpdatePlaceParams) error
	GetPlace(ctx context.Context, placeID string) (*domain.PlaceSummary, error)
	ListPlaces(ctx context.Context, p domain.ListPlacesParams) ([]domain.PlaceSummary, error)
	PlaceExists(ctx context.Context, placeID string) (bool, error)

	InsertProducts(ctx context.Context, placeID string, products []domain.InsertProductParams) error
	UpdateProductMenu(ctx context.Context, p domain.UpdateProductMenuParams) error
	SoftDeleteProduct(ctx context.Context, externalID, placeID string) error
	ListProducts(ctx context.Context, p domain.ListProductsParams) ([]domain.MenuItem, error)
	SearchProducts(ctx context.Context, p domain.SearchProductsParams) ([]domain.ProductSearchItem, error)
}
