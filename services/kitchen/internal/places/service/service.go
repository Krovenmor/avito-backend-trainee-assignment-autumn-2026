package service

import "avito-kitchen/services/kitchen/internal/places/domain"

import "context"

type Service interface {
	Authenticate(ctx context.Context, apiKey string) (*domain.PlaceAuth, error)
	RegisterPlace(ctx context.Context, in domain.RegisterPlaceInput) (*domain.RegisterPlaceOutput, error)
	UpdatePlace(ctx context.Context, in domain.UpdatePlaceInput) error
	AddMenuItems(ctx context.Context, placeID string, items []domain.MenuItemInput) ([]string, error)
	UpdateMenuItem(ctx context.Context, in domain.UpdateMenuItemInput) error
	DeleteMenuItem(ctx context.Context, placeID, externalID string) error
	GetPlace(ctx context.Context, placeID string) (*domain.PlaceSummary, error)
	ListPlaces(ctx context.Context, p domain.ListPlacesParams) ([]domain.PlaceSummary, error)
	ListProducts(ctx context.Context, p domain.ListProductsParams) ([]domain.MenuItem, error)
	SearchProducts(ctx context.Context, p domain.SearchProductsParams) ([]domain.ProductSearchItem, error)
}
