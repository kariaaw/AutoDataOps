package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const serviceName = "autodataops-control-plane"

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// NewHandler creates the HTTP API handler.
//
// Health endpoints intentionally expose no
// credentials or internal system configuration.
func NewHandler(checkReady func(context.Context) error) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		writeHealth(w, http.StatusOK, "alive")
	})

	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if checkReady == nil {
			writeHealth(w, http.StatusServiceUnavailable, "not_ready")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := checkReady(ctx); err != nil {
			writeHealth(w, http.StatusServiceUnavailable, "not_ready")
			return
		}

		writeHealth(w, http.StatusOK, "ready")
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")

		mux.ServeHTTP(w, r)
	})
}

func writeHealth(w http.ResponseWriter, code int, status string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	response := HealthResponse{
		Status:  status,
		Service: serviceName,
	}

	_ = json.NewEncoder(w).Encode(response)
}
