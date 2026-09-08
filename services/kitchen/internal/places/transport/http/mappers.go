package http

import (
	"net/http"

	"avito-kitchen/pkg/slice"
	"avito-kitchen/services/kitchen/internal/httpx"
	"avito-kitchen/services/kitchen/internal/places/domain"
)

func toRegisterPlaceInput(req registerPlaceRequest) domain.RegisterPlaceInput {
	return domain.RegisterPlaceInput{
		Name:        req.Name,
		Description: req.Description,
		City:        req.City,
		Address:     req.Address,
		Timezone:    req.Timezone,
		OpensAt:     req.OpensAt,
		ClosesAt:    req.ClosesAt,
		Type:        req.Type,
		WebhookURL:  req.WebhookURL,
	}
}

func toRegisterPlaceResponse(out *domain.RegisterPlaceOutput) registerPlaceResponse {
	return registerPlaceResponse{PlaceID: out.PlaceID, APIKey: out.APIKey}
}

func toUpdatePlaceInput(placeID string, req updatePlaceRequest) domain.UpdatePlaceInput {
	return domain.UpdatePlaceInput{
		PlaceID:     placeID,
		Name:        req.Name,
		Description: req.Description,
		City:        req.City,
		Address:     req.Address,
		Timezone:    req.Timezone,
		OpensAt:     req.OpensAt,
		ClosesAt:    req.ClosesAt,
		Type:        req.Type,
		WebhookURL:  req.WebhookURL,
		IsActive:    req.IsActive,
	}
}

func toMenuItemInput(req menuItemRequest) domain.MenuItemInput {
	return domain.MenuItemInput{
		ExternalID:    req.ExternalID,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
	}
}

func toMenuItemInputs(req []menuItemRequest) []domain.MenuItemInput {
	return slice.Map(req, toMenuItemInput)
}

func toMenuItemResponse(externalID string) menuItemResponse {
	return menuItemResponse{ExternalID: externalID}
}

func toMenuItemResponses(externalIDs []string) []menuItemResponse {
	return slice.Map(externalIDs, toMenuItemResponse)
}

func toUpdateMenuItemInput(placeID, externalID string, req updateMenuItemRequest) domain.UpdateMenuItemInput {
	return domain.UpdateMenuItemInput{
		PlaceID:     placeID,
		ExternalID:  externalID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		StockDelta:  req.StockDelta,
		Reason:      req.Reason,
	}
}

func toListPlacesParams(r *http.Request, limit, offset int32) domain.ListPlacesParams {
	return domain.ListPlacesParams{
		City:         httpx.OptionalQueryParam(r, queryParamCity),
		Type:         httpx.OptionalQueryParam(r, queryParamType),
		Search:       httpx.OptionalQueryParam(r, httpx.QueryParamSearch),
		MinRating:    httpx.OptionalQueryParam(r, queryParamMinRating),
		SortByRating: r.URL.Query().Get(queryParamSort) == sortByRating,
		Limit:        limit,
		Offset:       offset,
	}
}

func toPlaceListItemResponse(p domain.PlaceSummary) placeListItemResponse {
	return placeListItemResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		City:        p.City,
		Address:     p.Address,
		Timezone:    p.Timezone,
		OpensAt:     p.OpensAt,
		ClosesAt:    p.ClosesAt,
		Type:        p.Type,
		IsOpen:      p.IsOpen,
		Rating:      p.Rating,
		RatingCount: p.RatingCount,
	}
}

func toPlaceListItemResponses(places []domain.PlaceSummary) []placeListItemResponse {
	return slice.Map(places, toPlaceListItemResponse)
}

func toListProductsParams(r *http.Request, placeID string, limit, offset int32) domain.ListProductsParams {
	return domain.ListProductsParams{
		PlaceID: placeID,
		Search:  httpx.OptionalQueryParam(r, httpx.QueryParamSearch),
		Limit:   limit,
		Offset:  offset,
	}
}

func toProductListItemResponse(p domain.MenuItem) productListItemResponse {
	return productListItemResponse{
		ProductID:     p.ProductID,
		Name:          p.Name,
		Description:   p.Description,
		Price:         p.Price,
		StockQuantity: p.StockQuantity,
	}
}

func toProductListItemResponses(products []domain.MenuItem) []productListItemResponse {
	return slice.Map(products, toProductListItemResponse)
}

func toSearchProductsParams(r *http.Request, limit, offset int32) domain.SearchProductsParams {
	return domain.SearchProductsParams{
		City:   httpx.OptionalQueryParam(r, queryParamCity),
		Type:   httpx.OptionalQueryParam(r, queryParamType),
		Search: httpx.OptionalQueryParam(r, httpx.QueryParamSearch),
		Limit:  limit,
		Offset: offset,
	}
}

func toProductSearchItemResponse(item domain.ProductSearchItem) productSearchItemResponse {
	return productSearchItemResponse{
		ProductID:     item.ProductID,
		Name:          item.Name,
		Description:   item.Description,
		Price:         item.Price,
		StockQuantity: item.StockQuantity,
		Place: productPlaceResponse{
			PlaceID: item.Place.ID,
			Name:    item.Place.Name,
			City:    item.Place.City,
			Type:    item.Place.Type,
			IsOpen:  item.Place.IsOpen,
		},
	}
}

func toProductSearchItemResponses(items []domain.ProductSearchItem) []productSearchItemResponse {
	return slice.Map(items, toProductSearchItemResponse)
}
