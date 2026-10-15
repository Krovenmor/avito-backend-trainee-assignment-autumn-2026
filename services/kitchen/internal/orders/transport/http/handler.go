package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"avito-kitchen/services/kitchen/internal/httpx"
	"avito-kitchen/services/kitchen/internal/orders/domain"
	"avito-kitchen/services/kitchen/internal/orders/service"
	placesservice "avito-kitchen/services/kitchen/internal/places/service"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

const (
	pathParamID     = "id"
	pathParamUserID = "user_id"

	queryParamUserID = "user_id"

	idempotencyKeyHeader = "Idempotency-Key"

	heartbeatInterval = 20 * time.Second
	streamLifetime    = 30 * time.Minute
)

type Handler struct {
	svc      service.Service
	places   placesservice.Service
	validate *validator.Validate
	log      *zap.Logger
}

func NewHandler(svc service.Service, places placesservice.Service, validate *validator.Validate, log *zap.Logger) *Handler {
	return &Handler{svc: svc, places: places, validate: validate, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/orders", h.createOrder)
	mux.HandleFunc("GET /api/v1/orders/{id}", h.getOrder)
	mux.HandleFunc("GET /api/v1/orders/{id}/tracking", h.listOrderTracking)
	mux.HandleFunc("POST /api/v1/orders/{id}/cancel", h.cancelOrder)
	mux.HandleFunc("GET /api/v1/orders/{id}/stream", h.streamOrder)
	mux.HandleFunc("POST /api/v1/orders/{id}/rate", h.rateOrder)
	mux.HandleFunc("GET /api/v1/users/{user_id}/orders", h.listUserOrders)
	mux.HandleFunc("POST /api/v1/b2b/orders/{id}/delivered", h.requireAPIKey(h.markDelivered))
	mux.HandleFunc("POST /api/v1/b2b/orders/{id}/tracking", h.requireAPIKey(h.addTracking))
}

// @Summary Create an order
// @Description Reserves stock and creates a PENDING order. Requires an Idempotency-Key header.
// @Tags orders
// @Accept json
// @Produce json
// @Param Idempotency-Key header string true "Idempotency key (UUID)"
// @Param request body createOrderRequest true "Order to create"
// @Success 201 {object} createOrderResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} outOfStockResponse "Заведение закрыто, запрос с этим Idempotency-Key ещё выполняется, либо не хватает остатка — тогда в items перечислены проблемные позиции"
// @Failure 422 {object} httpx.ErrorResponse
// @Router /api/v1/orders [post]
func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get(idempotencyKeyHeader)
	if err := h.validate.Var(idempotencyKey, httpx.UUIDTag); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Idempotency-Key header must be a valid UUID")
		return
	}

	body, err := httpx.ReadBody(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "cannot read request body")
		return
	}

	req, err := httpx.UnmarshalAndValidate[createOrderRequest](h.validate, body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	out, err := h.svc.CreateOrder(r.Context(), toCreateOrderInput(idempotencyKey, hashBody(body), req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toCreateOrderResponse(out))
}

// @Summary Cancel an order
// @Description Cancels a PENDING order and returns the reserved stock. The place is not notified: if it accepts the order at the same moment, the platform records the cancellation and the place does not learn about it.
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param request body cancelOrderRequest true "Owner of the order"
// @Success 200
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /api/v1/orders/{id}/cancel [post]
func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
	if !ok {
		return
	}

	req, err := httpx.DecodeAndValidate[cancelOrderRequest](h.validate, r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.CancelOrder(r.Context(), orderID, httpx.NormalizeUUID(req.UserID)); err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary Stream order status
// @Description Server-sent events with the order status history and live updates.
// @Tags orders
// @Produce text/event-stream
// @Param id path string true "Order ID"
// @Param user_id query string true "Owner of the order"
// @Success 200 {object} orderEventResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/orders/{id}/stream [get]
func (h *Handler) streamOrder(w http.ResponseWriter, r *http.Request) {
	orderID, userID, ok := h.orderAndOwner(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), streamLifetime)
	defer cancel()

	events, err := h.svc.SubscribeOrderEvents(ctx, orderID, userID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	stream := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if err := stream.Flush(); err != nil {
		return
	}

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			data, err := json.Marshal(toOrderEventResponse(event))
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			_ = stream.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			_ = stream.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// @Summary List a user's orders
// @Description Order history for a user, newest first.
// @Tags orders
// @Produce json
// @Param user_id path string true "User ID"
// @Param limit query int false "Page size, default 20, max 100"
// @Param from query int false "Offset, default 0"
// @Success 200 {array} orderListItemResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /api/v1/users/{user_id}/orders [get]
func (h *Handler) listUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamUserID)
	if !ok {
		return
	}

	limit, offset := httpx.ParsePagination(r)
	orders, err := h.svc.ListUserOrders(r.Context(), toListUserOrdersParams(userID, limit, offset))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toOrderListItemResponses(orders))
}

// @Summary Mark an order as delivered
// @Description Moves an ACCEPTED order to COMPLETED. Idempotent for an order that is already COMPLETED.
// @Tags b2b/orders
// @Security ApiKeyAuth
// @Produce json
// @Param id path string true "Order ID"
// @Success 200
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /api/v1/b2b/orders/{id}/delivered [post]
func (h *Handler) markDelivered(w http.ResponseWriter, r *http.Request) {
	orderID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
	if !ok {
		return
	}

	placeID, ok := placeIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.svc.MarkDelivered(r.Context(), orderID, placeID); err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func hashBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// @Summary Rate a delivered order
// @Description Rates a COMPLETED order the user owns, once. The place rating is the average of such ratings, so only someone who actually received an order can affect it.
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param request body rateOrderRequest true "Owner of the order and the score from 1 to 5"
// @Success 200
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /api/v1/orders/{id}/rate [post]
func (h *Handler) rateOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
	if !ok {
		return
	}

	req, err := httpx.DecodeAndValidate[rateOrderRequest](h.validate, r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	in := domain.RateOrderInput{
		OrderID: orderID,
		UserID:  httpx.NormalizeUUID(req.UserID),
		Score:   req.Score,
	}
	if err := h.svc.RateOrder(r.Context(), in); err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary Report an order tracking event
// @Description The place reports its own progress on an accepted order. These statuses belong to the place, not to the platform: they are stored as history and forwarded to the user's SSE stream, but they never change order_status.
// @Tags b2b/orders
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param request body addTrackingRequest true "Tracking event"
// @Success 202
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/b2b/orders/{id}/tracking [post]
func (h *Handler) addTracking(w http.ResponseWriter, r *http.Request) {
	orderID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
	if !ok {
		return
	}

	placeID, ok := placeIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	req, err := httpx.DecodeAndValidate[addTrackingRequest](h.validate, r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.AddTracking(r.Context(), toAddTrackingInput(orderID, placeID, req)); err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// @Summary Get one order
// @Description The order with its items and the place name — everything an order page needs. user_id must match the owner, otherwise the order is reported as not found.
// @Tags orders
// @Produce json
// @Param id path string true "Order ID"
// @Param user_id query string true "Owner of the order"
// @Success 200 {object} orderResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/orders/{id} [get]
func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	orderID, userID, ok := h.orderAndOwner(w, r)
	if !ok {
		return
	}

	order, err := h.svc.GetOrder(r.Context(), orderID, userID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toOrderResponse(order))
}

// @Summary Order tracking history
// @Description Statuses reported by the place and, later, by delivery. These never change order_status; they are the observable track next to it. The SSE stream carries the same events live, this endpoint replays them.
// @Tags orders
// @Produce json
// @Param id path string true "Order ID"
// @Param user_id query string true "Owner of the order"
// @Success 200 {array} trackingEventResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/orders/{id}/tracking [get]
func (h *Handler) listOrderTracking(w http.ResponseWriter, r *http.Request) {
	orderID, userID, ok := h.orderAndOwner(w, r)
	if !ok {
		return
	}

	events, err := h.svc.ListOrderTracking(r.Context(), orderID, userID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toTrackingEventResponses(events))
}

func (h *Handler) orderAndOwner(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	orderID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
	if !ok {
		return "", "", false
	}
	userID, ok := httpx.UUIDQueryParam(h.validate, w, r, queryParamUserID)
	if !ok {
		return "", "", false
	}

	return orderID, userID, true
}
