package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestCreateOrder_IdempotentReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	key := uuid.NewString()
	body := map[string]any{
		"user_id":  userID,
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 2)},
	}

	var first createOrderResult
	createOrderRaw(t, key, body, http.StatusCreated, &first)

	var second createOrderResult
	createOrderRaw(t, key, body, http.StatusCreated, &second)

	if first.OrderID != second.OrderID {
		t.Fatalf("expected the same order_id on replay, got %s and %s", first.OrderID, second.OrderID)
	}

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 8 {
		t.Fatalf("expected stock to be reserved once (8 left), got %d", stock)
	}

	orders := userOrders(t, userID)
	if len(orders) != 1 {
		t.Fatalf("expected exactly one order in history, got %d", len(orders))
	}
}

func TestCreateOrder_IdempotencyKeyReusedWithDifferentBody(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	key := uuid.NewString()

	createOrderRaw(t, key, map[string]any{
		"user_id":  userID,
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
	}, http.StatusCreated, nil)

	createOrderRaw(t, key, map[string]any{
		"user_id":  userID,
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 3)},
	}, http.StatusUnprocessableEntity, nil)

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 9 {
		t.Fatalf("expected only the first order to reserve stock (9 left), got %d", stock)
	}
}

func TestCreateOrder_FailedAttemptReleasesIdempotencyKey(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 1},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	key := uuid.NewString()

	createOrderRaw(t, key, map[string]any{
		"user_id":  userID,
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 5)},
	}, http.StatusConflict, nil)

	var order createOrderResult
	createOrderRaw(t, key, map[string]any{
		"user_id":  userID,
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
	}, http.StatusCreated, &order)

	if order.OrderID == "" {
		t.Fatal("expected the same idempotency key to work after a failed attempt")
	}
}
