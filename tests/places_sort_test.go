package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestListPlaces_SortByRating(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	city := "SortCity-" + uuid.NewString()[:8]

	rate := func(score int) string {
		place := registerPlaceInCity(t, stub, city)
		addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
			{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
		})
		menu := placeMenu(t, place.PlaceID)

		userID := uuid.NewString()
		order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})
		waitOrderStatus(t, userID, order.OrderID, "accepted")
		doRequest(t, http.MethodPost, baseURL+"/api/v1/b2b/orders/"+order.OrderID+"/delivered",
			map[string]string{"X-API-Key": place.APIKey}, nil, http.StatusOK, nil)
		doRequest(t, http.MethodPost, baseURL+"/api/v1/orders/"+order.OrderID+"/rate",
			nil, map[string]any{"user_id": userID, "score": score}, http.StatusOK, nil)

		return place.PlaceID
	}

	low := rate(2)
	high := rate(5)

	var places []struct {
		ID     string  `json:"id"`
		Rating *string `json:"rating"`
	}
	doRequest(t, http.MethodGet, baseURL+"/api/v1/places?city="+city+"&sort=rating", nil, nil, http.StatusOK, &places)

	if len(places) < 2 {
		t.Fatalf("expected both places in %s, got %+v", city, places)
	}
	if places[0].ID != high || places[1].ID != low {
		t.Fatalf("expected the best rated place first, got %+v", places)
	}

	doRequest(t, http.MethodGet, baseURL+"/api/v1/places?city="+city+"&sort=garbage", nil, nil, http.StatusOK, &places)
	if len(places) < 2 {
		t.Fatalf("garbage sort must fall back to the default order, got %+v", places)
	}
}
