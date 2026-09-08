package tests

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type streamEvent struct {
	OrderID string `json:"order_id"`
	Source  string `json:"source"`
	Status  string `json:"status"`
}

func readStreamEvents(t *testing.T, url string, want int, timeout time.Duration) []streamEvent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new stream request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open stream: expected status 200, got %d", resp.StatusCode)
	}

	var events []streamEvent
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			t.Fatalf("decode stream event %q: %v", line, err)
		}
		events = append(events, event)

		if len(events) >= want {
			return events
		}
	}

	return events
}

func TestOrderStream_DeliversStatusHistory(t *testing.T) {
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

	streamURL := baseURL + "/api/v1/orders/" + order.OrderID + "/stream?user_id=" + userID
	events := readStreamEvents(t, streamURL, 2, 20*time.Second)

	if len(events) < 2 {
		t.Fatalf("expected at least pending and accepted events, got %+v", events)
	}
	if events[0].Status != "pending" {
		t.Fatalf("expected the first event to be pending, got %q", events[0].Status)
	}
	if events[1].Status != "accepted" {
		t.Fatalf("expected the second event to be accepted, got %q", events[1].Status)
	}
	for _, event := range events {
		if event.OrderID != order.OrderID {
			t.Fatalf("expected events for order %s, got %s", order.OrderID, event.OrderID)
		}
	}
}

func TestOrderStream_ReplaysHistoryAfterReconnect(t *testing.T) {
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
	waitOrderStatus(t, userID, order.OrderID, "accepted")

	streamURL := baseURL + "/api/v1/orders/" + order.OrderID + "/stream?user_id=" + userID
	events := readStreamEvents(t, streamURL, 2, 10*time.Second)

	if len(events) < 2 {
		t.Fatalf("expected the whole history on a late connect, got %+v", events)
	}
	if events[0].Status != "pending" || events[1].Status != "accepted" {
		t.Fatalf("expected pending then accepted, got %+v", events)
	}
}

func TestOrderStream_Validation(t *testing.T) {
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
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})

	t.Run("another user is not found", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodGet,
			baseURL+"/api/v1/orders/"+order.OrderID+"/stream?user_id="+uuid.NewString(), "", nil, http.StatusNotFound, nil)
	})

	t.Run("unknown order is not found", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodGet,
			baseURL+"/api/v1/orders/"+uuid.NewString()+"/stream?user_id="+userID, "", nil, http.StatusNotFound, nil)
	})

	t.Run("missing user_id rejected", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodGet,
			baseURL+"/api/v1/orders/"+order.OrderID+"/stream", "", nil, http.StatusBadRequest, nil)
	})

	t.Run("malformed order id rejected", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodGet,
			baseURL+"/api/v1/orders/not-a-uuid/stream?user_id="+userID, "", nil, http.StatusBadRequest, nil)
	})
}
