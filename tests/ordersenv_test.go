package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type webhookItem struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Quantity   int32  `json:"quantity"`
	Price      string `json:"price"`
}

type webhookRequest struct {
	OrderID     string        `json:"order_id"`
	PlaceID     string        `json:"place_id"`
	TotalAmount string        `json:"total_amount"`
	Items       []webhookItem `json:"items"`
}

type webhookStub struct {
	server *httptest.Server

	mu       sync.Mutex
	requests []webhookRequest
}

func newWebhookStub(t *testing.T, respond func(req webhookRequest) (int, string)) *webhookStub {
	t.Helper()

	stub := &webhookStub{}
	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req webhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		stub.mu.Lock()
		stub.requests = append(stub.requests, req)
		stub.mu.Unlock()

		status, body := respond(req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(stub.server.Close)

	return stub
}

func (s *webhookStub) calls() []webhookRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	calls := make([]webhookRequest, len(s.requests))
	copy(calls, s.requests)
	return calls
}

func acceptingWebhook(t *testing.T) *webhookStub {
	return newWebhookStub(t, func(webhookRequest) (int, string) {
		return http.StatusOK, `{"decision":"accepted"}`
	})
}

func failingWebhook(t *testing.T) *webhookStub {
	return newWebhookStub(t, func(webhookRequest) (int, string) {
		return http.StatusInternalServerError, `{}`
	})
}

func registerPlaceWithWebhook(t *testing.T, stub *webhookStub) placeRegistration {
	t.Helper()

	return registerTestPlace(t, map[string]any{
		"timezone":    "UTC",
		"opens_at":    "00:00:00",
		"closes_at":   "23:59:59",
		"webhook_url": stub.server.URL,
	})
}

func registerPlaceInCity(t *testing.T, stub *webhookStub, city string) placeRegistration {
	t.Helper()

	return registerTestPlace(t, map[string]any{
		"city":        city,
		"timezone":    "UTC",
		"opens_at":    "00:00:00",
		"closes_at":   "23:59:59",
		"webhook_url": stub.server.URL,
	})
}

func closedWindow() (string, string) {
	if time.Now().UTC().Hour() >= 2 {
		return "00:00:00", "01:00:00"
	}
	return "23:00:00", "23:59:59"
}

type menuItem struct {
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	Price         string `json:"price"`
	StockQuantity int32  `json:"stock_quantity"`
}

func placeMenu(t *testing.T, placeID string) []menuItem {
	t.Helper()

	var menu []menuItem
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/"+placeID+"/menu", "", nil, http.StatusOK, &menu)
	return menu
}

func menuItemByName(t *testing.T, placeID, name string) menuItem {
	t.Helper()

	for _, item := range placeMenu(t, placeID) {
		if item.Name == name {
			return item
		}
	}

	t.Fatalf("menu item %q not found in place %s", name, placeID)
	return menuItem{}
}

func productStock(t *testing.T, placeID, productID string) int32 {
	t.Helper()

	for _, item := range placeMenu(t, placeID) {
		if item.ProductID == productID {
			return item.StockQuantity
		}
	}

	t.Fatalf("product %s not found in menu of place %s", productID, placeID)
	return 0
}

type unavailableItemBody struct {
	ProductID string `json:"product_id"`
	Requested int32  `json:"requested"`
	Available int32  `json:"available"`
}

type outOfStockBody struct {
	Error string                `json:"error"`
	Items []unavailableItemBody `json:"items"`
}

type createOrderResult struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

func createOrderRaw(t *testing.T, idempotencyKey string, body any, wantStatus int, out any) {
	t.Helper()

	headers := map[string]string{}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	doRequest(t, http.MethodPost, baseURL+"/api/v1/orders", headers, body, wantStatus, out)
}

func createOrder(t *testing.T, userID, placeID string, items []map[string]any) createOrderResult {
	t.Helper()

	var result createOrderResult
	createOrderRaw(t, uuid.NewString(), map[string]any{
		"user_id":  userID,
		"place_id": placeID,
		"items":    items,
	}, http.StatusCreated, &result)

	return result
}

func orderItem(productID string, quantity int32) map[string]any {
	return map[string]any{"product_id": productID, "quantity": quantity}
}

type userOrder struct {
	OrderID     string `json:"order_id"`
	PlaceID     string `json:"place_id"`
	Status      string `json:"status"`
	TotalAmount string `json:"total_amount"`
}

func userOrders(t *testing.T, userID string) []userOrder {
	t.Helper()

	var orders []userOrder
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/users/"+userID+"/orders", "", nil, http.StatusOK, &orders)
	return orders
}

func orderStatus(t *testing.T, userID, orderID string) string {
	t.Helper()

	for _, order := range userOrders(t, userID) {
		if order.OrderID == orderID {
			return order.Status
		}
	}

	t.Fatalf("order %s not found in history of user %s", orderID, userID)
	return ""
}

func waitOrderStatus(t *testing.T, userID, orderID, want string) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		last = orderStatus(t, userID, orderID)
		if last == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("order %s: expected status %q, last seen %q", orderID, want, last)
}

func rejectionBody(reason string, externalIDs ...string) string {
	ids, err := json.Marshal(externalIDs)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"decision":"rejected","reason":%q,"unavailable_external_ids":%s}`, reason, ids)
}
