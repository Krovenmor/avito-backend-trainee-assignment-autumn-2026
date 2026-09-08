package http

import (
	"errors"
	"net/http"

	"avito-kitchen/services/kitchen/internal/httpx"
	"avito-kitchen/services/kitchen/internal/places/domain"

	"go.uber.org/zap"
)

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrDuplicateExternalID):
		httpx.WriteError(w, http.StatusConflict, "external_id already exists for this place")
	case errors.Is(err, domain.ErrStockConflict):
		httpx.WriteError(w, http.StatusConflict, "stock_quantity would go negative")
	case errors.Is(err, domain.ErrInvalidPrice):
		httpx.WriteError(w, http.StatusBadRequest, "price must be non-negative")
	default:
		h.log.Error("unhandled service error",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
