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
		headers       map[string]string
		body          any
		expectStatus  int
		expectHeaders map[string]string
		expectBody    string
	}{
		{
			name:         "Default application/json when headers are omitted",
			status:       http.StatusOK,
			headers:      nil,
			body:         map[string]string{"message": "hello"},
			expectStatus: http.StatusOK,
			expectHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectBody: `{"message":"hello"}`,
		},
		{
			name:   "Custom response headers and custom Content-Type with raw text body",
			status: http.StatusOK,
			headers: map[string]string{
				"Content-Type":  "text/plain",
				"X-Cache":       "HIT",
				"Cache-Control": "public, max-age=3600",
			},
			body:         "raw text payload",
			expectStatus: http.StatusOK,
			expectHeaders: map[string]string{
				"Content-Type":  "text/plain",
				"X-Cache":       "HIT",
				"Cache-Control": "public, max-age=3600",
			},
			expectBody: "raw text payload",
		},
		{
			name:   "Sanitizes CRLF in headers to prevent response splitting",
			status: http.StatusCreated,
			headers: map[string]string{
				"X-Trace\r\nInjected": "trace-value\r\nInjected",
			},
			body:         map[string]bool{"ok": true},
			expectStatus: http.StatusCreated,
			expectHeaders: map[string]string{
				"X-TraceInjected": "trace-valueInjected",
				"Content-Type":    "application/json",
			},
			expectBody: `{"ok":true}`,
		},
		{
			name:         "No Content 204 writes no body or Content-Type",
			status:       http.StatusNoContent,
			headers:      map[string]string{"X-Deleted": "true"},
			body:         nil,
			expectStatus: http.StatusNoContent,
			expectHeaders: map[string]string{
				"X-Deleted": "true",
			},
			expectBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			err := xrespond.Write(w, tt.status, tt.headers, tt.body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			for k, expectedVal := range tt.expectHeaders {
				if actualVal := w.Header().Get(k); actualVal != expectedVal {
					t.Errorf("expected header %q to be %q, got %q", k, expectedVal, actualVal)
				}
			}

			trimmed := strings.TrimSpace(w.Body.String())
			if trimmed != tt.expectBody {
				t.Errorf("expected body %q, got %q", tt.expectBody, trimmed)
			}
		})
	}
}
