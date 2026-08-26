package xrespond_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/parser"
	"github.com/ju4n97/hclapi/internal/steps/xrespond"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cfg          *parser.RespondStepConfig
		lastResult   any
		expectStatus int
		expectBody   string
	}{
		{
			name: "Static body write",
			cfg: &parser.RespondStepConfig{
				Status: http.StatusOK,
				Body:   new(`{"message":"hello"}`),
			},
			lastResult:   map[string]string{"should": "ignore"},
			expectStatus: http.StatusOK,
			expectBody:   `{"message":"hello"}`,
		},
		{
			name: "Fallback to lastResult serialization",
			cfg: &parser.RespondStepConfig{
				Status: http.StatusCreated,
				Body:   nil,
			},
			lastResult:   map[string]int{"count": 42},
			expectStatus: http.StatusCreated,
			expectBody:   `{"count":42}`,
		},
		{
			name: "Empty response when both are nil",
			cfg: &parser.RespondStepConfig{
				Status: http.StatusNoContent,
				Body:   nil,
			},
			lastResult:   nil,
			expectStatus: http.StatusNoContent,
			expectBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			err := xrespond.Write(w, tt.cfg, tt.lastResult)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			trimmed := strings.TrimSpace(w.Body.String())
			if trimmed != tt.expectBody {
				t.Errorf("expected body %s, got %s", tt.expectBody, trimmed)
			}
		})
	}
}
