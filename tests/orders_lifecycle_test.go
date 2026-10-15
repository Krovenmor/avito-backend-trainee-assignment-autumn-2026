package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCancelOrder(t *testing.T) {
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
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 3)})

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 7 {
		t.Fatalf("expected stock 7 after reservation, got %d", stock)
	}

	t.Run("wrong user is not found", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/"+order.OrderID+"/cancel", "",
			map[string]any{"user_id": uuid.NewString()}, http.StatusNotFound, nil)
	})

	t.Run("unknown order is not found", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/"+uuid.NewString()+"/cancel", "",
			map[string]any{"user_id": userID}, http.StatusNotFound, nil)
	})

	t.Run("malformed order id rejected", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/not-a-uuid/cancel", "",
			map[string]any{"user_id": userID}, http.StatusBadRequest, nil)
	})

	t.Run("owner cancels and stock returns", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/"+order.OrderID+"/cancel", "",
			map[string]any{"user_id": userID}, http.StatusOK, nil)

		if status := orderStatus(t, userID, order.OrderID); status != "cancelled" {
			t.Fatalf("expected status cancelled, got %q", status)
		}
		if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 10 {
			t.Fatalf("expected stock to be returned (10), got %d", stock)
		}
	})

	t.Run("second cancel conflicts", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/"+order.OrderID+"/cancel", "",
			map[string]any{"user_id": userID}, http.StatusConflict, nil)

		if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 10 {
			t.Fatalf("expected stock to stay 10 after a repeated cancel, got %d", stock)
		}
	})
}

func TestCancelOrder_StopsCallingThePlace(t *testing.T) {
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
	waitWebhookCalls(t, stub, 1)

	doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/"+order.OrderID+"/cancel", "",
		map[string]any{"user_id": userID}, http.StatusOK, nil)
	calls := len(stub.calls())

	time.Sleep(5 * time.Second)
	if extra := len(stub.calls()) - calls; extra > 0 {
		t.Fatalf("expected the place to stop receiving a cancelled order, got %d more webhook calls", extra)
	}
}

func TestMarkDelivered(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	accepting := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, accepting)
	other := registerPlaceWithWebhook(t, accepting)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})
	deliveredURL := baseURL + "/api/v1/b2b/orders/" + order.OrderID + "/delivered"

	t.Run("without api key unauthorized", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, deliveredURL, "", nil, http.StatusUnauthorized, nil)
	})

	t.Run("invalid api key unauthorized", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, deliveredURL, "not-a-real-key", nil, http.StatusUnauthorized, nil)
	})

	waitOrderStatus(t, userID, order.OrderID, "accepted")

	t.Run("another place cannot deliver", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, deliveredURL, other.APIKey, nil, http.StatusNotFound, nil)
	})

	t.Run("owner delivers", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, deliveredURL, place.APIKey, nil, http.StatusOK, nil)

		if status := orderStatus(t, userID, order.OrderID); status != "completed" {
			t.Fatalf("expected status completed, got %q", status)
		}
	})

	t.Run("repeated delivery is idempotent", func(t *testing.T) {
		doJSON(t, httpClient, http.MethodPost, deliveredURL, place.APIKey, nil, http.StatusOK, nil)
	})
}

func TestMarkDelivered_PendingOrderConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := failingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
	})
	menu := placeMenu(t, place.PlaceID)

	order := createOrder(t, uuid.NewString(), place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})

	doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/orders/"+order.OrderID+"/delivered",
		place.APIKey, nil, http.StatusConflict, nil)
}

func TestWebhookRejection_ReturnsStock(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := newWebhookStub(t, func(webhookRequest) (int, string) {
		return http.StatusOK, rejectionBody("too_busy")
	})
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 4)})

	waitOrderStatus(t, userID, order.OrderID, "rejected")

	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 10 {
		t.Fatalf("expected stock to be returned after rejection (10), got %d", stock)
	}
}

func TestWebhookRejection_OutOfStockZeroesStock(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := newWebhookStub(t, func(webhookRequest) (int, string) {
		return http.StatusOK, rejectionBody("out_of_stock", "sku-1")
	})
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Gone", "price": "10.00", "stock_quantity": 10},
		{"external_id": "sku-2", "name": "Still there", "price": "10.00", "stock_quantity": 8},
	})
	menu := placeMenu(t, place.PlaceID)

	var gone, present menuItem
	for _, item := range menu {
		if item.Name == "Gone" {
			gone = item
		} else {
			present = item
		}
	}

	userID := uuid.NewString()
	order := createOrder(t, userID, place.PlaceID, []map[string]any{
		orderItem(gone.ProductID, 3),
		orderItem(present.ProductID, 2),
	})

	waitOrderStatus(t, userID, order.OrderID, "rejected")

	if stock := productStock(t, place.PlaceID, gone.ProductID); stock != 0 {
		t.Fatalf("expected the unavailable product stock to be zeroed, got %d", stock)
	}
	if stock := productStock(t, place.PlaceID, present.ProductID); stock != 8 {
		t.Fatalf("expected the remaining product stock to be returned (8), got %d", stock)
	}
}

func TestWebhookRejection_IgnoresExternalIDsOutsideTheOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := newWebhookStub(t, func(webhookRequest) (int, string) {
		return http.StatusOK, rejectionBody("out_of_stock", "sku-1", "sku-2")
	})
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Ordered", "price": "10.00", "stock_quantity": 10},
		{"external_id": "sku-2", "name": "Untouched", "price": "10.00", "stock_quantity": 8},
	})

	ordered := menuItemByName(t, place.PlaceID, "Ordered")
	untouched := menuItemByName(t, place.PlaceID, "Untouched")

	userID := uuid.NewString()
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(ordered.ProductID, 3)})

	waitOrderStatus(t, userID, order.OrderID, "rejected")

	if stock := productStock(t, place.PlaceID, ordered.ProductID); stock != 0 {
		t.Fatalf("expected the ordered product stock to be zeroed, got %d", stock)
	}
	if stock := productStock(t, place.PlaceID, untouched.ProductID); stock != 8 {
		t.Fatalf("expected a product outside the order to keep its stock (8), got %d", stock)
	}
}

func TestWebhookInvalidResponse_KeepsOrderPending(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := newWebhookStub(t, func(webhookRequest) (int, string) {
		return http.StatusOK, `{"decision":"maybe"}`
	})
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 10},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})

	if status := orderStatus(t, userID, order.OrderID); status != "pending" {
		t.Fatalf("expected the order to stay pending after an invalid webhook response, got %q", status)
	}
	if stock := productStock(t, place.PlaceID, menu[0].ProductID); stock != 9 {
		t.Fatalf("expected the stock to stay reserved while retrying (9), got %d", stock)
	}
}

func TestListUserOrders(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	stub := failingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 20},
	})
	menu := placeMenu(t, place.PlaceID)

	userID := uuid.NewString()
	first := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 1)})
	second := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(menu[0].ProductID, 2)})

	orders := userOrders(t, userID)
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders in history, got %d", len(orders))
	}
	if orders[0].OrderID != second.OrderID {
		t.Fatalf("expected the newest order first, got %s", orders[0].OrderID)
	}
	if orders[0].TotalAmount != "20.00" {
		t.Fatalf("expected total_amount 20.00 for 2 items at 10.00, got %q", orders[0].TotalAmount)
	}

	var paged []userOrder
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/users/"+userID+"/orders?limit=1", "", nil, http.StatusOK, &paged)
	if len(paged) != 1 {
		t.Fatalf("expected 1 order with limit=1, got %d", len(paged))
	}

	var offsetPage []userOrder
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/users/"+userID+"/orders?limit=1&from=1", "", nil, http.StatusOK, &offsetPage)
	if len(offsetPage) != 1 || offsetPage[0].OrderID != first.OrderID {
		t.Fatalf("expected the older order on the second page, got %+v", offsetPage)
	}

	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/users/not-a-uuid/orders", "", nil, http.StatusBadRequest, nil)
}
