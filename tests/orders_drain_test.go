package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestShutdown_LeavesOrderPendingForAnotherInstance(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	release := make(chan struct{})
	stub := newWebhookStub(t, func(webhookRequest) (int, string) {
		<-release
		return http.StatusInternalServerError, `{}`
	})
	t.Cleanup(func() { close(release) })

	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	menu := placeMenu(t, place.PlaceID)

	secondURL, stopSecond, err := startKitchenServer(envDBConn, envRedis)
	if err != nil {
		t.Fatalf("start the second kitchen instance: %v", err)
	}

	userID := uuid.NewString()
	var order createOrderResult
	doRequest(t, http.MethodPost, secondURL+"/api/v1/orders",
		map[string]string{"Idempotency-Key": uuid.NewString()},
		map[string]any{
			"user_id":  userID,
			"place_id": place.PlaceID,
			"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
		}, http.StatusCreated, &order)

	waitWebhookCalls(t, stub, 1)

	start := time.Now()
	stopSecond()
	t.Logf("shutdown took %s", time.Since(start))

	if status := orderStatus(t, userID, order.OrderID); status != "pending" {
		t.Fatalf("expected the order to stay pending after shutdown, got %q", status)
	}
}
