package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPendingOrder_IsClaimedByOneInstanceOnly(t *testing.T) {
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

	userID := uuid.NewString()
	createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})

	waitWebhookCalls(t, stub, 1)

	_, stopSecond, err := startKitchenServer(envDBConn, envRedis)
	if err != nil {
		t.Fatalf("start the second kitchen instance: %v", err)
	}
	t.Cleanup(stopSecond)

	time.Sleep(3 * time.Second)

	if calls := len(stub.calls()); calls != 1 {
		t.Fatalf("expected the order to stay leased to the first instance (1 webhook call), got %d", calls)
	}
}

func waitWebhookCalls(t *testing.T, stub *webhookStub, want int) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if len(stub.calls()) >= want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("expected at least %d webhook calls, got %d", want, len(stub.calls()))
}
