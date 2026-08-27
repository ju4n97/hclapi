package xrespond_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/steps/xrespond"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		status        int
		evaluatedBody any
		lastResult    any
		expectStatus  int
		expectBody    string
	}{
		{
			name:          "Explicit evaluated body takes precedence over lastResult",
			status:        http.StatusOK,
			evaluatedBody: map[string]string{"message": "hello"},
			lastResult:    map[string]string{"ignored": "true"},
			expectStatus:  http.StatusOK,
			expectBody:    `{"message":"hello"}`,
		},
		{
			name:          "Fallback to lastResult when evaluatedBody is nil",
			status:        http.StatusCreated,
			evaluatedBody: nil,
			lastResult:    map[string]int{"count": 42},
			expectStatus:  http.StatusCreated,
			expectBody:    `{"count":42}`,
		},
		{
			name:          "Empty body when both evaluatedBody and lastResult are nil",
			status:        http.StatusNoContent,
			evaluatedBody: nil,
			lastResult:    nil,
			expectStatus:  http.StatusNoContent,
			expectBody:    "",
		},
		{
			name:          "Serializes custom error payload with 404 status",
			status:        http.StatusNotFound,
			evaluatedBody: map[string]string{"error": "user not found"},
			lastResult:    nil,
			expectStatus:  http.StatusNotFound,
			expectBody:    `{"error":"user not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			err := xrespond.Write(w, tt.status, tt.evaluatedBody, tt.lastResult)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", ct)
			}

			trimmed := strings.TrimSpace(w.Body.String())
			if trimmed != tt.expectBody {
				t.Errorf("expected body %q, got %q", tt.expectBody, trimmed)
			}
		})
	}
}
