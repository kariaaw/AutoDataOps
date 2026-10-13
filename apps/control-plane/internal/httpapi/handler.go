package httpapi

import (
	"encoding/json"
	"net/http"
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
func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		writeHealth(w, "alive")
	})

	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		// Bootstrap readiness only.
		// Dependency checks will be added later.
		writeHealth(w, "ready")
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")

		mux.ServeHTTP(w, r)
	})
}

func writeHealth(w http.ResponseWriter, status string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status:  status,
		Service: serviceName,
	}

	_ = json.NewEncoder(w).Encode(response)
}
