package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessStates(t *testing.T) {
	failingCheck := func(context.Context) error {
		return errors.New("database unavailable")
	}

	successfulCheck := func(context.Context) error {
		return nil
	}

	tests := []struct {
		name       string
		path       string
		check      func(context.Context) error
		wantCode   int
		wantStatus string
	}{
		{"missing dependency", "/health/ready", nil, 503, "not_ready"},
		{"database failure", "/health/ready", failingCheck, 503, "not_ready"},
		{"database healthy", "/health/ready", successfulCheck, 200, "ready"},
		{"liveness independent", "/health/live", failingCheck, 200, "alive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			NewHandler(tt.check).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantCode {
				t.Fatalf("expected HTTP %d, got %d", tt.wantCode, recorder.Code)
			}

			var response HealthResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("invalid response JSON: %v", err)
			}

			if response.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, response.Status)
			}
		})
	}
}
