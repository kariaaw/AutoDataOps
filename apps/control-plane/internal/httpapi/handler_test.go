package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus string
	}{
		{
			name:       "liveness",
			path:       "/health/live",
			wantStatus: "alive",
		},
		{
			name:       "readiness",
			path:       "/health/ready",
			wantStatus: "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			recorder := httptest.NewRecorder()

			NewHandler().ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf(
					"expected HTTP 200, got %d",
					recorder.Code,
				)
			}

			var response HealthResponse

			if err := json.Unmarshal(
				recorder.Body.Bytes(),
				&response,
			); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if response.Status != tt.wantStatus {
				t.Errorf(
					"expected status %q, got %q",
					tt.wantStatus,
					response.Status,
				)
			}

			if response.Service != serviceName {
				t.Errorf(
					"unexpected service: %q",
					response.Service,
				)
			}

			if recorder.Header().Get("Cache-Control") != "no-store" {
				t.Error("missing no-store cache policy")
			}

			if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("missing nosniff security header")
			}

			if recorder.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Error("unexpected content type")
			}
		})
	}
}

func TestUnknownEndpoint(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/unknown",
		nil,
	)

	recorder := httptest.NewRecorder()

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected HTTP 404, got %d",
			recorder.Code,
		)
	}
}

func TestUnsupportedMethod(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/health/live",
		nil,
	)

	recorder := httptest.NewRecorder()

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected HTTP 405, got %d",
			recorder.Code,
		)
	}
}
