package http

import (
	"errors"
	"net/http"

	"avito-kitchen/services/kitchen/internal/httpx"
	"avito-kitchen/services/kitchen/internal/orders/domain"

	"go.uber.org/zap"
)

const outOfStockMessage = "not enough stock"

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var outOfStock *domain.OutOfStockError
	if errors.As(err, &outOfStock) {
		httpx.WriteJSON(w, http.StatusConflict, toOutOfStockResponse(outOfStock.Items))
		return
	}

	switch {
	case errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrPlaceClosed):
		httpx.WriteError(w, http.StatusConflict, "place is closed")
	case errors.Is(err, domain.ErrOutOfStock):
		httpx.WriteError(w, http.StatusConflict, "not enough stock")
	case errors.Is(err, domain.ErrUnknownProducts):
		httpx.WriteError(w, http.StatusBadRequest, "some products do not belong to this place")
	case errors.Is(err, domain.ErrAlreadyRated):
		httpx.WriteError(w, http.StatusConflict, "order is already rated")
	case errors.Is(err, domain.ErrStatusConflict):
		httpx.WriteError(w, http.StatusConflict, "order status does not allow this transition")
	case errors.Is(err, domain.ErrIdempotencyInProgress):
		httpx.WriteError(w, http.StatusConflict, "another request with this idempotency key is in progress")
	case errors.Is(err, domain.ErrIdempotencyMismatch):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "idempotency key reused with a different body")
	default:
		h.log.Error("unhandled service error",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
