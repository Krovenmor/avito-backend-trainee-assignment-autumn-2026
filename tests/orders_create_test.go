package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestCreateOrder_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
		{"external_id": "sku-2", "name": "Pepperoni", "price": "12.50", "stock_quantity": 4},
	})

	menu := placeMenu(t, place.PlaceID)
	if len(menu) != 2 {
		t.Fatalf("expected 2 menu items, got %d", len(menu))
	}

	userID := uuid.NewString()
	order := createOrder(t, userID, place.PlaceID, []map[string]any{
		orderItem(menu[0].ProductID, 2),
		orderItem(menu[1].ProductID, 1),
	})

	if order.OrderID == "" {
		t.Fatal("expected order_id in response")
	}
	if order.Status != "pending" {
		t.Fatalf("expected status pending, got %q", order.Status)
	}

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != menu[0].StockQuantity-2 {
		t.Fatalf("expected stock %d after reservation, got %d", menu[0].StockQuantity-2, stock)
	}
	if stock := productStock(t, place.PlaceID, menu[1].ProductID); stock != menu[1].StockQuantity-1 {
		t.Fatalf("expected stock %d after reservation, got %d", menu[1].StockQuantity-1, stock)
	}

	waitOrderStatus(t, userID, order.OrderID, "accepted")

	calls := stub.calls()
	if len(calls) == 0 {
		t.Fatal("expected the place webhook to be called")
	}
	if calls[0].OrderID != order.OrderID {
		t.Fatalf("expected webhook for order %s, got %s", order.OrderID, calls[0].OrderID)
	}
	if len(calls[0].Items) != 2 {
		t.Fatalf("expected 2 items in webhook payload, got %d", len(calls[0].Items))
	}
	for _, item := range calls[0].Items {
		if item.ExternalID == "" {
			t.Fatalf("expected external_id in webhook payload, got %+v", item)
		}
	}
}

func TestCreateOrder_AcceptsUppercaseUUIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()

	var order createOrderResult
	createOrderRaw(t, strings.ToUpper(uuid.NewString()), map[string]any{
		"user_id":  strings.ToUpper(userID),
		"place_id": strings.ToUpper(place.PlaceID),
		"items":    []map[string]any{orderItem(strings.ToUpper(menu[0].ProductID), 1)},
	}, http.StatusCreated, &order)

	waitOrderStatus(t, userID, order.OrderID, "accepted")

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 4 {
		t.Fatalf("expected the reservation to hit the same product (4), got %d", stock)
	}
}

func TestCreateOrder_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	menu := placeMenu(t, place.PlaceID)
	userID := uuid.NewString()

	validBody := map[string]any{
		"user_id":  userID,
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
	}

	t.Run("missing idempotency key", func(t *testing.T) {
		createOrderRaw(t, "", validBody, http.StatusBadRequest, nil)
	})

	t.Run("idempotency key is not a uuid", func(t *testing.T) {
		createOrderRaw(t, "not-a-uuid", validBody, http.StatusBadRequest, nil)
	})

	t.Run("user_id is not a uuid", func(t *testing.T) {
		createOrderRaw(t, uuid.NewString(), map[string]any{
			"user_id":  "not-a-uuid",
			"place_id": place.PlaceID,
			"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
		}, http.StatusBadRequest, nil)
	})

	t.Run("empty items rejected", func(t *testing.T) {
		createOrderRaw(t, uuid.NewString(), map[string]any{
			"user_id":  userID,
			"place_id": place.PlaceID,
			"items":    []map[string]any{},
		}, http.StatusBadRequest, nil)
	})

	t.Run("zero quantity rejected", func(t *testing.T) {
		createOrderRaw(t, uuid.NewString(), map[string]any{
			"user_id":  userID,
			"place_id": place.PlaceID,
			"items":    []map[string]any{orderItem(menu[0].ProductID, 0)},
		}, http.StatusBadRequest, nil)
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		createOrderRaw(t, uuid.NewString(), map[string]any{
			"user_id":  userID,
			"place_id": place.PlaceID,
			"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
			"surprise": "value",
		}, http.StatusBadRequest, nil)
	})

	t.Run("unknown product rejected", func(t *testing.T) {
		createOrderRaw(t, uuid.NewString(), map[string]any{
			"user_id":  userID,
			"place_id": place.PlaceID,
			"items":    []map[string]any{orderItem(uuid.NewString(), 1)},
		}, http.StatusBadRequest, nil)
	})

	t.Run("unknown place is not found", func(t *testing.T) {
		createOrderRaw(t, uuid.NewString(), map[string]any{
			"user_id":  userID,
			"place_id": uuid.NewString(),
			"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
		}, http.StatusNotFound, nil)
	})
}

func TestCreateOrder_ProductFromAnotherPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	placeA := registerPlaceWithWebhook(t, stub)
	placeB := registerPlaceWithWebhook(t, stub)

	addTestMenuItems(t, placeA.PlaceID, placeA.APIKey, []map[string]any{
		{"external_id": "sku-a", "name": "From A", "price": "10.00", "stock_quantity": 5},
	})
	addTestMenuItems(t, placeB.PlaceID, placeB.APIKey, []map[string]any{
		{"external_id": "sku-b", "name": "From B", "price": "10.00", "stock_quantity": 5},
	})

	menuA := placeMenu(t, placeA.PlaceID)
	menuB := placeMenu(t, placeB.PlaceID)

	createOrderRaw(t, uuid.NewString(), map[string]any{
		"user_id":  uuid.NewString(),
		"place_id": placeA.PlaceID,
		"items": []map[string]any{
			orderItem(menuA[0].ProductID, 1),
			orderItem(menuB[0].ProductID, 1),
		},
	}, http.StatusBadRequest, nil)

	if stock := productStock(t, placeA.PlaceID, menuA[0].ProductID); stock != 5 {
		t.Fatalf("expected stock of place A to stay 5 after a rejected order, got %d", stock)
	}
}

func TestCreateOrder_OutOfStockSumsDuplicatedItems(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 4},
	})
	product := placeMenu(t, place.PlaceID)[0]

	var rejection outOfStockBody
	createOrderRaw(t, uuid.NewString(), map[string]any{
		"user_id":  uuid.NewString(),
		"place_id": place.PlaceID,
		"items": []map[string]any{
			orderItem(product.ProductID, 2),
			orderItem(product.ProductID, 3),
		},
	}, http.StatusConflict, &rejection)

	if len(rejection.Items) != 1 {
		t.Fatalf("expected duplicated lines to collapse into one reported item, got %+v", rejection.Items)
	}
	if got := rejection.Items[0]; got.Requested != 5 || got.Available != 4 {
		t.Fatalf("expected requested 5 (2+3) against available 4, got %+v", got)
	}

	if stock := productStock(t, place.PlaceID, product.ProductID); stock != 4 {
		t.Fatalf("expected the stock to stay untouched at 4, got %d", stock)
	}
}

func TestCreateOrder_OutOfStock(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Scarce", "price": "10.00", "stock_quantity": 2},
		{"external_id": "sku-2", "name": "Plenty", "price": "10.00", "stock_quantity": 10},
	})
	scarce := menuItemByName(t, place.PlaceID, "Scarce")
	plenty := menuItemByName(t, place.PlaceID, "Plenty")

	var rejection outOfStockBody
	createOrderRaw(t, uuid.NewString(), map[string]any{
		"user_id":  uuid.NewString(),
		"place_id": place.PlaceID,
		"items": []map[string]any{
			orderItem(plenty.ProductID, 1),
			orderItem(scarce.ProductID, 3),
		},
	}, http.StatusConflict, &rejection)

	if len(rejection.Items) != 1 {
		t.Fatalf("expected exactly the scarce item to be reported, got %+v", rejection.Items)
	}
	if got := rejection.Items[0]; got.ProductID != scarce.ProductID || got.Requested != 3 || got.Available != 2 {
		t.Fatalf("expected the scarce product with requested 3 and available 2, got %+v", got)
	}

	for _, item := range placeMenu(t, place.PlaceID) {
		expected := int32(2)
		if item.Name == "Plenty" {
			expected = 10
		}
		if item.StockQuantity != expected {
			t.Fatalf("expected stock of %q to stay %d after a failed order, got %d", item.Name, expected, item.StockQuantity)
		}
	}
}

func TestCreateOrder_PlaceClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	opensAt, closesAt := closedWindow()
	place := registerTestPlace(t, map[string]any{
		"timezone":    "UTC",
		"opens_at":    opensAt,
		"closes_at":   closesAt,
		"webhook_url": stub.server.URL,
	})
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	menu := placeMenu(t, place.PlaceID)

	createOrderRaw(t, uuid.NewString(), map[string]any{
		"user_id":  uuid.NewString(),
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
	}, http.StatusConflict, nil)
}

func TestCreateOrder_InactivePlace(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 5},
	})
	menu := placeMenu(t, place.PlaceID)

	doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID, place.APIKey,
		map[string]any{"is_active": false}, http.StatusOK, nil)

	createOrderRaw(t, uuid.NewString(), map[string]any{
		"user_id":  uuid.NewString(),
		"place_id": place.PlaceID,
		"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
	}, http.StatusConflict, nil)
}

func TestCreateOrder_ConcurrentOrdersNeverOversell(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	const (
		scarceStock = 3
		plentyStock = 100
		orders      = 20
	)

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Scarce", "price": "10.00", "stock_quantity": scarceStock},
		{"external_id": "sku-2", "name": "Plenty", "price": "10.00", "stock_quantity": plentyStock},
	})
	scarce := menuItemByName(t, place.PlaceID, "Scarce")
	plenty := menuItemByName(t, place.PlaceID, "Plenty")

	start := make(chan struct{})
	statuses := make(chan int, orders)
	failures := make(chan error, orders)
	var wg sync.WaitGroup
	for i := range orders {
		items := []map[string]any{orderItem(scarce.ProductID, 1), orderItem(plenty.ProductID, 1)}
		if i%2 == 1 {
			items[0], items[1] = items[1], items[0]
		}
		body, err := json.Marshal(map[string]any{
			"user_id":  uuid.NewString(),
			"place_id": place.PlaceID,
			"items":    items,
		})
		if err != nil {
			t.Fatalf("marshal order: %v", err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, err := postOrder(body)
			if err != nil {
				failures <- err
				return
			}
			statuses <- status
		}()
	}
	close(start)
	wg.Wait()
	close(statuses)
	close(failures)

	for err := range failures {
		t.Fatalf("post order: %v", err)
	}

	created := 0
	for status := range statuses {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
		default:
			t.Fatalf("expected only 201 or 409 from concurrent orders, got %d", status)
		}
	}

	if created != scarceStock {
		t.Fatalf("expected exactly %d orders to get the scarce item, got %d", scarceStock, created)
	}
	if stock := productStock(t, place.PlaceID, scarce.ProductID); stock != 0 {
		t.Fatalf("expected the scarce item to be sold out, got stock %d", stock)
	}
	if stock := productStock(t, place.PlaceID, plenty.ProductID); stock != plentyStock-scarceStock {
		t.Fatalf("expected rejected orders to reserve nothing, got plenty stock %d", stock)
	}
}

func postOrder(body []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/orders", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
