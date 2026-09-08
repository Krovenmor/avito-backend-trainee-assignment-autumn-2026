package tests

import (
	"net/http"
	"testing"
)

func TestAddMenuItems_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, nil)

	t.Run("empty array rejected", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu", place.APIKey,
			[]map[string]any{}, http.StatusBadRequest, nil)
	})

	t.Run("negative price rejected", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu", place.APIKey,
			[]map[string]any{{"external_id": "bad-price", "name": "Item", "price": "-1.00", "stock_quantity": 1}},
			http.StatusBadRequest, nil)
	})

	t.Run("missing external_id rejected", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu", place.APIKey,
			[]map[string]any{{"name": "Item", "price": "1.00", "stock_quantity": 1}},
			http.StatusBadRequest, nil)
	})

	t.Run("duplicate external_id within place conflicts", func(t *testing.T) {
		addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
			{"external_id": "dup-1", "name": "Item", "price": "1.00", "stock_quantity": 1},
		})
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu", place.APIKey,
			[]map[string]any{{"external_id": "dup-1", "name": "Item Again", "price": "2.00", "stock_quantity": 1}},
			http.StatusConflict, nil)
	})
}

func TestAddMenuItems_KeepsInternalIDInternal(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, nil)

	var resp []map[string]any
	doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu", place.APIKey,
		[]map[string]any{{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5}},
		http.StatusCreated, &resp)

	if len(resp) != 1 {
		t.Fatalf("expected one confirmation, got %d", len(resp))
	}
	if resp[0]["external_id"] != "sku-1" {
		t.Fatalf("expected the confirmation to echo external_id, got %+v", resp[0])
	}
	if _, exposed := resp[0]["product_id"]; exposed {
		t.Fatalf("expected the b2b response to keep the internal product id internal, got %+v", resp[0])
	}
}

func TestUpdateMenuItem_Logic(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, nil)
	items := addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	externalID := items[0].ExternalID

	t.Run("negative stock_delta without reason rejected", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/"+externalID, place.APIKey,
			map[string]any{"stock_delta": -1}, http.StatusBadRequest, nil)
	})

	t.Run("stock_delta that would go negative conflicts", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/"+externalID, place.APIKey,
			map[string]any{"stock_delta": -100, "reason": "spoilage"}, http.StatusConflict, nil)
	})

	t.Run("negative stock_delta with reason applies atomically", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/"+externalID, place.APIKey,
			map[string]any{"stock_delta": -3, "reason": "spoilage"}, http.StatusOK, nil)

		if stock := menuItemByName(t, place.PlaceID, "Margherita").StockQuantity; stock != 2 {
			t.Fatalf("expected stock_quantity 2 after 5-3, got %d", stock)
		}
	})

	t.Run("unknown external_id not found", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/does-not-exist", place.APIKey,
			map[string]any{"name": "New Name"}, http.StatusNotFound, nil)
	})

	t.Run("external_id from another place not found", func(t *testing.T) {
		otherPlace := registerTestPlace(t, nil)
		addTestMenuItems(t, otherPlace.PlaceID, otherPlace.APIKey, []map[string]any{
			{"external_id": "other-sku", "name": "Other Item", "price": "1.00", "stock_quantity": 1},
		})
		doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/other-sku", place.APIKey,
			map[string]any{"name": "New Name"}, http.StatusNotFound, nil)
	})
}

func TestDeleteMenuItem_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, nil)
	items := addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	externalID := items[0].ExternalID

	doJSON(t, httpClient, http.MethodDelete, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/"+externalID, place.APIKey, nil, http.StatusOK, nil)
	doJSON(t, httpClient, http.MethodDelete, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/"+externalID, place.APIKey, nil, http.StatusOK, nil)

	doJSON(t, httpClient, http.MethodDelete, baseURL+"/api/v1/b2b/places/"+place.PlaceID+"/menu/never-existed", place.APIKey, nil, http.StatusNotFound, nil)

	for _, item := range placeMenu(t, place.PlaceID) {
		if item.Name == "Margherita" {
			t.Fatalf("expected the deleted product to be absent from the menu listing, got %+v", item)
		}
	}
}

func TestListMenu_SearchAndPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, nil)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita Pizza", "price": "10.00", "stock_quantity": 5},
		{"external_id": "sku-2", "name": "Pepperoni Pizza", "price": "12.00", "stock_quantity": 3},
		{"external_id": "sku-3", "name": "Caesar Salad", "price": "8.00", "stock_quantity": 7},
	})

	var all []struct {
		Name string `json:"name"`
	}
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID+"/menu", "", nil, http.StatusOK, &all)
	if len(all) != 3 {
		t.Fatalf("expected 3 items total, got %d", len(all))
	}

	var searched []struct {
		Name string `json:"name"`
	}
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID+"/menu?search=pizza", "", nil, http.StatusOK, &searched)
	if len(searched) != 2 {
		t.Fatalf("expected 2 items matching 'pizza', got %d: %+v", len(searched), searched)
	}

	var paged []struct {
		Name string `json:"name"`
	}
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID+"/menu?limit=1", "", nil, http.StatusOK, &paged)
	if len(paged) != 1 {
		t.Fatalf("expected 1 item with limit=1, got %d", len(paged))
	}
}

func TestListMenu_HidesDeactivatedPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, map[string]any{"name": "Скоро закроется", "city": "Kazan"})
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})

	if items := placeMenu(t, place.PlaceID); len(items) != 1 {
		t.Fatalf("expected the menu of an active place to be readable, got %+v", items)
	}

	doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID, place.APIKey,
		map[string]any{"is_active": false}, http.StatusOK, nil)

	doRequest(t, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID+"/menu", nil, nil, http.StatusNotFound, nil)
	doRequest(t, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID, nil, nil, http.StatusNotFound, nil)
}
