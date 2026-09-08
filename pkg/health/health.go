package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const (
	LivenessPath  = "/healthz"
	ReadinessPath = "/readyz"

	checkTimeout = 2 * time.Second

	statusOK          = "ok"
	statusUnavailable = "unavailable"
)

type Check func(ctx context.Context) error

type response struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

func Register(mux *http.ServeMux, checks map[string]Check) {
	mux.HandleFunc("GET "+LivenessPath, func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, response{Status: statusOK})
	})

	mux.HandleFunc("GET "+ReadinessPath, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)
		defer cancel()

		failed := make(map[string]string)
		for name, check := range checks {
			if err := check(ctx); err != nil {
				failed[name] = err.Error()
			}
		}

		if len(failed) > 0 {
			write(w, http.StatusServiceUnavailable, response{Status: statusUnavailable, Checks: failed})
			return
		}

		write(w, http.StatusOK, response{Status: statusOK})
	})
}

func write(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
