package http

import (
	"context"
	"errors"
	"net/http"

	"avito-kitchen/services/kitchen/internal/httpx"
	placesdomain "avito-kitchen/services/kitchen/internal/places/domain"
)

const (
	apiKeyHeader = "X-API-Key"
)

type contextKey int

const (
	placeIDContextKey contextKey = iota
)

func (h *Handler) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(apiKeyHeader)
		if apiKey == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing X-API-Key")
			return
		}

		auth, err := h.places.Authenticate(r.Context(), apiKey)
		if err != nil {
			if errors.Is(err, placesdomain.ErrNotFound) {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid api key")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), placeIDContextKey, auth.ID)))
	}
}

func placeIDFromContext(ctx context.Context) (string, bool) {
	placeID, ok := ctx.Value(placeIDContextKey).(string)
	return placeID, ok
}
