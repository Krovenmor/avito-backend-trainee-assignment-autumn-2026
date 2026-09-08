package http

import (
	"context"
	"errors"
	"net/http"

	"avito-kitchen/services/kitchen/internal/httpx"
	"avito-kitchen/services/kitchen/internal/places/domain"
)

const (
	apiKeyHeader = "X-API-Key"
)

type contextKey int

const (
	authContextKey contextKey = iota
)

func (h *Handler) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(apiKeyHeader)
		if apiKey == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing X-API-Key")
			return
		}

		auth, err := h.svc.Authenticate(r.Context(), apiKey)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid api key")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), authContextKey, auth)))
	}
}

func (h *Handler) requireOwnPlace(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth, ok := r.Context().Value(authContextKey).(*domain.PlaceAuth)
		if !ok {
			httpx.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		placeID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
		if !ok {
			return
		}

		if placeID != auth.ID {
			httpx.WriteError(w, http.StatusForbidden, "place id mismatch")
			return
		}

		next(w, r)
	}
}
