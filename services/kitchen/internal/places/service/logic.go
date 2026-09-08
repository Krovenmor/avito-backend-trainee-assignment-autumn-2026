package service

import (
	"avito-kitchen/services/kitchen/internal/places/domain"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type svc struct {
	repo Repository
	log  *zap.Logger
}

func New(repo Repository, log *zap.Logger) Service {
	return &svc{repo: repo, log: log}
}

func hashAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

func generateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *svc) Authenticate(ctx context.Context, apiKey string) (*domain.PlaceAuth, error) {
	return s.repo.GetPlaceByAPIKeyHash(ctx, hashAPIKey(apiKey))
}

func (s *svc) RegisterPlace(ctx context.Context, in domain.RegisterPlaceInput) (*domain.RegisterPlaceOutput, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	placeID := uuid.NewString()
	err = s.repo.InsertPlace(ctx, domain.Place{
		ID:          placeID,
		Name:        in.Name,
		Description: in.Description,
		City:        in.City,
		Address:     in.Address,
		Timezone:    in.Timezone,
		OpensAt:     in.OpensAt,
		ClosesAt:    in.ClosesAt,
		Type:        in.Type,
		WebhookURL:  in.WebhookURL,
		APIKeyHash:  hashAPIKey(apiKey),
	})
	if err != nil {
		return nil, err
	}

	s.log.Debug("place registered",
		zap.String("place_id", placeID),
		zap.String("name", in.Name),
		zap.String("city", in.City),
	)

	return &domain.RegisterPlaceOutput{PlaceID: placeID, APIKey: apiKey}, nil
}

func (s *svc) UpdatePlace(ctx context.Context, in domain.UpdatePlaceInput) error {
	return s.repo.UpdatePlace(ctx, domain.UpdatePlaceParams{
		ID:          in.PlaceID,
		Name:        in.Name,
		Description: in.Description,
		City:        in.City,
		Address:     in.Address,
		Timezone:    in.Timezone,
		OpensAt:     in.OpensAt,
		ClosesAt:    in.ClosesAt,
		Type:        in.Type,
		WebhookURL:  in.WebhookURL,
		IsActive:    in.IsActive,
	})
}

func (s *svc) AddMenuItems(ctx context.Context, placeID string, items []domain.MenuItemInput) ([]string, error) {
	params := make([]domain.InsertProductParams, len(items))
	externalIDs := make([]string, len(items))
	for i, item := range items {
		params[i] = domain.InsertProductParams{
			ID:            uuid.NewString(),
			ExternalID:    item.ExternalID,
			Name:          item.Name,
			Description:   item.Description,
			Price:         item.Price,
			StockQuantity: item.StockQuantity,
		}
		externalIDs[i] = item.ExternalID
	}

	if err := s.repo.InsertProducts(ctx, placeID, params); err != nil {
		return nil, err
	}
	return externalIDs, nil
}

func (s *svc) UpdateMenuItem(ctx context.Context, in domain.UpdateMenuItemInput) error {
	err := s.repo.UpdateProductMenu(ctx, domain.UpdateProductMenuParams{
		ExternalID:  in.ExternalID,
		PlaceID:     in.PlaceID,
		Name:        in.Name,
		Description: in.Description,
		Price:       in.Price,
		StockDelta:  in.StockDelta,
	})
	if err != nil {
		return err
	}

	if in.StockDelta != nil && *in.StockDelta < 0 {
		s.log.Info("stock written off",
			zap.String("place_id", in.PlaceID),
			zap.String("external_id", in.ExternalID),
			zap.Int32("stock_delta", *in.StockDelta),
			zap.String("reason", in.Reason),
		)
	}

	return nil
}

func (s *svc) DeleteMenuItem(ctx context.Context, placeID, externalID string) error {
	return s.repo.SoftDeleteProduct(ctx, externalID, placeID)
}

func (s *svc) GetPlace(ctx context.Context, placeID string) (*domain.PlaceSummary, error) {
	return s.repo.GetPlace(ctx, placeID)
}

func (s *svc) ListPlaces(ctx context.Context, p domain.ListPlacesParams) ([]domain.PlaceSummary, error) {
	return s.repo.ListPlaces(ctx, p)
}

func (s *svc) ListProducts(ctx context.Context, p domain.ListProductsParams) ([]domain.MenuItem, error) {
	exists, err := s.repo.PlaceExists(ctx, p.PlaceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}
	return s.repo.ListProducts(ctx, p)
}

func (s *svc) SearchProducts(ctx context.Context, p domain.SearchProductsParams) ([]domain.ProductSearchItem, error) {
	return s.repo.SearchProducts(ctx, p)
}
