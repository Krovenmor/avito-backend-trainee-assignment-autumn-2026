package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestRestartedInstance_ResumesItsOwnOrderWithoutWaitingForTheLease(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := newWebhookStub(t, func(webhookRequest) (int, string) {
		return http.StatusInternalServerError, `{}`
	})

	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	menu := placeMenu(t, place.PlaceID)

	port, err := freePort()
	if err != nil {
		t.Fatalf("pick a port: %v", err)
	}

	ownURL, stopOwn, err := startKitchenServerOnPort(envDBConn, envRedis, port)
	if err != nil {
		t.Fatalf("start the dedicated kitchen instance: %v", err)
	}

	userID := uuid.NewString()
	var order createOrderResult
	doRequest(t, http.MethodPost, ownURL+"/api/v1/orders",
		map[string]string{"Idempotency-Key": uuid.NewString()},
		map[string]any{
			"user_id":  userID,
			"place_id": place.PlaceID,
			"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
		}, http.StatusCreated, &order)

	waitWebhookCalls(t, stub, 1)
	stopOwn()

	before := len(stub.calls())

	_, stopRestarted, err := startKitchenServerOnPort(envDBConn, envRedis, port)
	if err != nil {
		t.Fatalf("restart the kitchen instance on the same port: %v", err)
	}
	t.Cleanup(stopRestarted)

	waitWebhookCalls(t, stub, before+1)

	if status := orderStatus(t, userID, order.OrderID); status != "pending" {
		t.Fatalf("expected the order to stay pending while the place keeps failing, got %q", status)
	}
}
