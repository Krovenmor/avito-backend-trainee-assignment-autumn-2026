package tests

import (
	"net/http"
	"testing"
)

func TestPlacesHappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, map[string]any{"name": "Pizza Place"})
	if place.PlaceID == "" || place.APIKey == "" {
		t.Fatalf("expected place_id and api_key in response, got %+v", place)
	}

	menuResp := addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.50", "stock_quantity": 5},
		{"external_id": "sku-2", "name": "Pepperoni", "price": "12.00", "stock_quantity": 3},
	})
	if len(menuResp) != 2 {
		t.Fatalf("expected 2 menu items, got %d", len(menuResp))
	}
	externalID := menuResp[0].ExternalID
	productID := menuItemByName(t, place.PlaceID, "Margherita").ProductID

	var placesResp []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places", "", nil, http.StatusOK, &placesResp)

	found := false
	for _, p := range placesResp {
		if p.ID == place.PlaceID && p.Name == "Pizza Place" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected registered place to appear in GET /places, got %+v", placesResp)
	}

	var menuListResp []struct {
		ProductID     string `json:"product_id"`
		StockQuantity int32  `json:"stock_quantity"`
	}
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID+"/menu", "", nil, http.StatusOK, &menuListResp)
	if len(menuListResp) != 2 {
		t.Fatalf("expected 2 menu items in listing, got %d", len(menuListResp))
	}

	doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/"+externalID, place.APIKey, map[string]any{
		"stock_delta": 10,
	}, http.StatusOK, nil)

	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID+"/menu", "", nil, http.StatusOK, &menuListResp)
	for _, p := range menuListResp {
		if p.ProductID == productID && p.StockQuantity != 15 {
			t.Fatalf("expected stock_quantity 15 after +10 delta, got %d", p.StockQuantity)
		}
	}

	doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID, place.APIKey, map[string]any{
		"name": "Best Pizza Place",
	}, http.StatusOK, nil)

	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places", "", nil, http.StatusOK, &placesResp)
	renamed := false
	for _, p := range placesResp {
		if p.ID == place.PlaceID && p.Name == "Best Pizza Place" {
			renamed = true
		}
	}
	if !renamed {
		t.Fatalf("expected place name to be updated, got %+v", placesResp)
	}

	doJSON(t, httpClient, http.MethodDelete, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/"+externalID, place.APIKey, nil, http.StatusOK, nil)

	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID+"/menu", "", nil, http.StatusOK, &menuListResp)
	for _, p := range menuListResp {
		if p.ProductID == productID {
			t.Fatalf("expected deleted product to be absent from menu listing, got %+v", menuListResp)
		}
	}
}
