package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

type productSearchItem struct {
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	Price         string `json:"price"`
	StockQuantity int32  `json:"stock_quantity"`
	Place         struct {
		PlaceID string `json:"place_id"`
		Name    string `json:"name"`
		City    string `json:"city"`
		Type    string `json:"type"`
	} `json:"place"`
}

func searchProducts(t *testing.T, query string) []productSearchItem {
	t.Helper()

	var items []productSearchItem
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/products"+query, "", nil, http.StatusOK, &items)
	return items
}

func TestSearchProducts_FindsDishAcrossPlaces(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	dish := "Пицца " + uuid.NewString()[:8]

	first := registerTestPlace(t, map[string]any{"name": "Первая", "city": "Kazan", "type": "restaurant"})
	addTestMenuItems(t, first.PlaceID, first.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": dish, "price": "590.00", "stock_quantity": 5},
		{"external_id": "sku-2", "name": "Суп " + uuid.NewString()[:8], "price": "200.00", "stock_quantity": 5},
	})

	second := registerTestPlace(t, map[string]any{"name": "Вторая", "city": "Omsk", "type": "cafe"})
	addTestMenuItems(t, second.PlaceID, second.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": dish, "price": "450.00", "stock_quantity": 3},
	})

	items := searchProducts(t, "?search="+urlQueryEscape(dish))
	if len(items) != 2 {
		t.Fatalf("expected the dish to be found in both places, got %d items: %+v", len(items), items)
	}

	places := map[string]productSearchItem{}
	for _, item := range items {
		if item.Name != dish {
			t.Fatalf("expected only the searched dish, got %q", item.Name)
		}
		places[item.Place.PlaceID] = item
	}

	if _, ok := places[first.PlaceID]; !ok {
		t.Fatal("expected the first place in the results")
	}
	if got := places[second.PlaceID]; got.Place.City != "Omsk" || got.Place.Type != "cafe" || got.Price != "450.00" {
		t.Fatalf("expected the second place with its own city, type and price, got %+v", got)
	}
}

func TestSearchProducts_FiltersByCityAndType(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	dish := "Рамен " + uuid.NewString()[:8]

	inCity := registerTestPlace(t, map[string]any{"name": "Нужная", "city": "Perm", "type": "cafe"})
	addTestMenuItems(t, inCity.PlaceID, inCity.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": dish, "price": "300.00", "stock_quantity": 2},
	})

	other := registerTestPlace(t, map[string]any{"name": "Другая", "city": "Tver", "type": "shop"})
	addTestMenuItems(t, other.PlaceID, other.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": dish, "price": "300.00", "stock_quantity": 2},
	})

	items := searchProducts(t, "?search="+urlQueryEscape(dish)+"&city=Perm")
	if len(items) != 1 || items[0].Place.PlaceID != inCity.PlaceID {
		t.Fatalf("expected only the Perm place, got %+v", items)
	}

	items = searchProducts(t, "?search="+urlQueryEscape(dish)+"&type=shop")
	if len(items) != 1 || items[0].Place.PlaceID != other.PlaceID {
		t.Fatalf("expected only the shop, got %+v", items)
	}
}

func TestSearchProducts_HidesDeletedAndInactive(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	dish := "Донер " + uuid.NewString()[:8]

	deleted := registerTestPlace(t, map[string]any{"name": "Со снятой позицией", "city": "Sochi"})
	addTestMenuItems(t, deleted.PlaceID, deleted.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": dish, "price": "250.00", "stock_quantity": 4},
	})
	doJSON(t, httpClient, http.MethodDelete,
		baseURL+"/api/v1/b2b/places/"+deleted.PlaceID+"/menu/sku-1", deleted.APIKey, nil, http.StatusOK, nil)

	inactive := registerTestPlace(t, map[string]any{"name": "Отключённая", "city": "Sochi"})
	addTestMenuItems(t, inactive.PlaceID, inactive.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": dish, "price": "250.00", "stock_quantity": 4},
	})
	doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+inactive.PlaceID,
		inactive.APIKey, map[string]any{"is_active": false}, http.StatusOK, nil)

	if items := searchProducts(t, "?search="+urlQueryEscape(dish)); len(items) != 0 {
		t.Fatalf("expected deleted products and inactive places to stay out of search, got %+v", items)
	}
}
