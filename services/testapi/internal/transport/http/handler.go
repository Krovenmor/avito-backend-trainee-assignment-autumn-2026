package http

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"avito-kitchen/services/testapi/internal/client"
	testapiconfig "avito-kitchen/services/testapi/internal/config"
	"avito-kitchen/services/testapi/internal/stock"

	"go.uber.org/zap"
)

const (
	WebhookPath = "/webhook/orders"

	decisionAccepted = "accepted"
	decisionRejected = "rejected"
	reasonOutOfStock = "out_of_stock"

	trackingCooking = "cooking"
	trackingReady   = "ready"

	codeKitchenQueue = "KITCHEN_QUEUE"
	codeKitchenDone  = "KITCHEN_DONE"

	deliveryTimeout = 10 * time.Second
	trackingTimeout = 5 * time.Second
	cookingDelay    = 300 * time.Millisecond
)

type Handler struct {
	stock         *stock.Stock
	client        *client.Client
	credentials   *client.Credentials
	deliveryDelay time.Duration
	log           *zap.Logger

	mu        sync.Mutex
	decisions map[string]decisionResponse
}

func NewHandler(
	stock *stock.Stock,
	apiClient *client.Client,
	credentials *client.Credentials,
	conf testapiconfig.Config,
	log *zap.Logger,
) *Handler {
	return &Handler{
		stock:         stock,
		client:        apiClient,
		credentials:   credentials,
		deliveryDelay: conf.DeliveryDelay,
		log:           log,
		decisions:     make(map[string]decisionResponse),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST "+WebhookPath, h.handleOrder)
}

func (h *Handler) handleOrder(w http.ResponseWriter, r *http.Request) {
	var order orderPayload
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "cannot decode order", http.StatusBadRequest)
		return
	}

	decision, decided := h.decide(order)
	if !decided && decision.Decision == decisionAccepted {
		go h.deliverLater(order.OrderID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(decision)
}

func (h *Handler) decide(order orderPayload) (decisionResponse, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if decision, exists := h.decisions[order.OrderID]; exists {
		h.log.Info("repeated webhook for a known order",
			zap.String("order_id", order.OrderID), zap.String("decision", decision.Decision))
		return decision, true
	}

	decision := decisionResponse{Decision: decisionAccepted}
	if unavailable := h.stock.Reserve(toRequestedLines(order.Items)); len(unavailable) > 0 {
		decision = decisionResponse{
			Decision:               decisionRejected,
			Reason:                 reasonOutOfStock,
			UnavailableExternalIDs: unavailable,
		}
	}

	h.decisions[order.OrderID] = decision
	h.log.Info("decided on order",
		zap.String("order_id", order.OrderID), zap.String("decision", decision.Decision))

	return decision, false
}

func (h *Handler) reportTracking(orderID, externalCode, status string) {
	_, apiKey := h.credentials.Get()
	if apiKey == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), trackingTimeout)
	defer cancel()

	tracking := client.TrackingRequest{ExternalCode: externalCode, Status: status, OccurredAt: time.Now().UTC()}
	if err := h.client.ReportTracking(ctx, orderID, apiKey, tracking); err != nil {
		h.log.Warn("report tracking", zap.String("order_id", orderID), zap.Error(err))
		return
	}

	h.log.Info("reported tracking", zap.String("order_id", orderID), zap.String("status", status))
}

func (h *Handler) deliverLater(orderID string) {
	h.reportTracking(orderID, codeKitchenQueue, trackingCooking)
	time.Sleep(cookingDelay)
	h.reportTracking(orderID, codeKitchenDone, trackingReady)

	time.Sleep(h.deliveryDelay)

	_, apiKey := h.credentials.Get()
	if apiKey == "" {
		h.log.Error("cannot mark order delivered without an api key", zap.String("order_id", orderID))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), deliveryTimeout)
	defer cancel()

	if err := h.client.MarkDelivered(ctx, orderID, apiKey); err != nil {
		h.log.Error("mark order delivered", zap.String("order_id", orderID), zap.Error(err))
		return
	}

	h.log.Info("order delivered", zap.String("order_id", orderID))
}
