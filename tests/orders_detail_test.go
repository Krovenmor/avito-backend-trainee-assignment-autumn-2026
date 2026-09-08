package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderDetail struct {
	OrderID     string `json:"order_id"`
	PlaceID     string `json:"place_id"`
	PlaceName   string `json:"place_name"`
	Status      string `json:"status"`
	TotalAmount string `json:"total_amount"`
	Items       []struct {
		ProductID  string `json:"product_id"`
		ExternalID string `json:"external_id"`
		Name       string `json:"name"`
		Quantity   int32  `json:"quantity"`
		Price      string `json:"price"`
	} `json:"items"`
}

type trackingEvent struct {
	Source       string    `json:"source"`
	ExternalCode string    `json:"external_code"`
	Status       string    `json:"status"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func TestGetOrder_ReturnsItemsAndHidesForeignOrders(t *testing.T) {
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
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 2)})

	var detail orderDetail
	doRequest(t, http.MethodGet, baseURL+"/api/v1/orders/"+order.OrderID+"?user_id="+userID,
		nil, nil, http.StatusOK, &detail)

	if len(detail.Items) != 1 {
		t.Fatalf("expected one item in the order, got %+v", detail)
	}
	if detail.Items[0].Name != "Margherita" || detail.Items[0].Quantity != 2 {
		t.Fatalf("unexpected item: %+v", detail.Items[0])
	}
	if detail.Items[0].ExternalID != "sku-1" {
		t.Fatalf("expected the place article, got %q", detail.Items[0].ExternalID)
	}
	if detail.TotalAmount != "20.00" {
		t.Fatalf("expected the total to be 20.00, got %q", detail.TotalAmount)
	}
	if detail.PlaceName == "" {
		t.Fatal("expected the place name so the client does not have to resolve it")
	}

	doRequest(t, http.MethodGet, baseURL+"/api/v1/orders/"+order.OrderID+"?user_id="+uuid.NewString(),
		nil, nil, http.StatusNotFound, nil)
}

func TestOrderTracking_HistoryIsReadableByTheOwner(t *testing.T) {
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
			"status":        "cooking",
			"occurred_at":   time.Now().UTC().Format(time.RFC3339),
		}, http.StatusAccepted, nil)

	var events []trackingEvent
	doRequest(t, http.MethodGet, baseURL+"/api/v1/orders/"+order.OrderID+"/tracking?user_id="+userID,
		nil, nil, http.StatusOK, &events)

	if len(events) != 1 {
		t.Fatalf("expected one tracking event, got %+v", events)
	}
	if events[0].Source != "place" || events[0].Status != "cooking" || events[0].ExternalCode != "KITCHEN_QUEUE" {
		t.Fatalf("unexpected tracking event: %+v", events[0])
	}

	doRequest(t, http.MethodGet, baseURL+"/api/v1/orders/"+order.OrderID+"/tracking?user_id="+uuid.NewString(),
		nil, nil, http.StatusNotFound, nil)
}

func TestGetPlace_ReturnsDetailsAndOpenState(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)

	var detail struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		City   string `json:"city"`
		IsOpen bool   `json:"is_open"`
	}
	doRequest(t, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID, nil, nil, http.StatusOK, &detail)

	if detail.ID != place.PlaceID || detail.Name == "" || detail.City == "" {
		t.Fatalf("unexpected place: %+v", detail)
	}
	if !detail.IsOpen {
		t.Fatal("expected a place open around the clock to report is_open")
	}

	doRequest(t, http.MethodGet, baseURL+"/api/v1/places/"+uuid.NewString(), nil, nil, http.StatusNotFound, nil)
}
