package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestCancelOrder_ReturnsStockForDuplicatedItems(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := failingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	order := createOrder(t, userID, place.PlaceID, []map[string]any{
		orderItem(menu[0].ProductID, 2),
		orderItem(menu[0].ProductID, 3),
	})

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 5 {
		t.Fatalf("expected both lines to be reserved (5 left), got %d", stock)
	}

	doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/"+order.OrderID+"/cancel", "",
		map[string]any{"user_id": userID}, http.StatusOK, nil)

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 10 {
		t.Fatalf("expected the whole reservation to be returned (10), got %d", stock)
	}
}
