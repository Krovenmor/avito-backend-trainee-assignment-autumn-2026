package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOrderTracking_ReachesTheStreamAndStaysWithinThePlace(t *testing.T) {
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
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})

	trackingURL := baseURL + "/api/v1/b2b/orders/" + order.OrderID + "/tracking"
	tracking := map[string]any{
		"external_code": "KITCHEN_QUEUE",
		"status":        "cooking",
		"occurred_at":   time.Now().UTC().Format(time.RFC3339),
	}
	doRequest(t, http.MethodPost, trackingURL,
		map[string]string{"X-API-Key": place.APIKey}, tracking, http.StatusAccepted, nil)

	streamURL := baseURL + "/api/v1/orders/" + order.OrderID + "/stream?user_id=" + userID
	events := readStreamEvents(t, streamURL, 3, 20*time.Second)

	var fromPlace, fromPlatform []streamEvent
	for _, event := range events {
		switch event.Source {
		case "place":
			fromPlace = append(fromPlace, event)
		case "platform":
			fromPlatform = append(fromPlatform, event)
		default:
			t.Fatalf("unexpected event source: %+v", event)
		}
	}

	if len(fromPlace) != 1 || fromPlace[0].Status != "cooking" {
		t.Fatalf("expected exactly one cooking event from the place, got %+v", events)
	}
	if len(fromPlatform) != 2 {
		t.Fatalf("expected pending and accepted from the platform, got %+v", events)
	}
	if fromPlatform[0].Status != "pending" || fromPlatform[1].Status != "accepted" {
		t.Fatalf("expected the platform track to stay ordered, got %+v", fromPlatform)
	}

	foreign := registerPlaceWithWebhook(t, stub)
	doRequest(t, http.MethodPost, trackingURL,
		map[string]string{"X-API-Key": foreign.APIKey}, tracking, http.StatusNotFound, nil)
}

func TestOrderTracking_RejectsUnknownStatus(t *testing.T) {
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
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})

	doRequest(t, http.MethodPost, baseURL+"/api/v1/b2b/orders/"+order.OrderID+"/tracking",
		map[string]string{"X-API-Key": place.APIKey},
		map[string]any{
			"external_code": "KITCHEN_QUEUE",
			"status":        "completed",
			"occurred_at":   time.Now().UTC().Format(time.RFC3339),
		}, http.StatusBadRequest, nil)
}
