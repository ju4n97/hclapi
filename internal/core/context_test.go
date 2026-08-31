package core_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
)

func TestNewContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		url         string
		headers     map[string]string
		pathParams  map[string]string
		bodyJSON    string
		expectError bool
		validate    func(t *testing.T, ctx *core.Context)
	}{
		{
			name:   "Parses query parameters and lowercases headers",
			method: http.MethodGet,
			url:    "/users?limit=10&status=active",
			headers: map[string]string{
				"X-Trace-ID":   "trace-123",
				"Content-Type": "application/json",
			},
			expectError: false,
			validate: func(t *testing.T, ctx *core.Context) {
				if ctx.Request.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", ctx.Request.Method)
				}
				if ctx.Request.Query["limit"] != "10" || ctx.Request.Query["status"] != "active" {
					t.Errorf("unexpected query params: %+v", ctx.Request.Query)
				}
				if ctx.Request.Headers["x-trace-id"] != "trace-123" {
					t.Errorf("expected lowercase header key, got: %+v", ctx.Request.Headers)
				}
			},
		},
		{
			name:        "Returns error on malformed JSON payload",
			method:      http.MethodPost,
			url:         "/sanitize",
			bodyJSON:    `{"tags": ["golang" "testing"]}`, // Missing comma
			expectError: true,
		},
		{
			name:        "Decodes valid JSON body payload",
			method:      http.MethodPost,
			url:         "/items",
			bodyJSON:    `{"title": "Test Item", "count": 5}`,
			expectError: false,
			validate: func(t *testing.T, ctx *core.Context) {
				body, ok := ctx.Request.Body.(map[string]any)
				if !ok {
					t.Fatalf("expected map[string]any body, got %T", ctx.Request.Body)
				}
				if body["title"] != "Test Item" {
					t.Errorf("expected 'Test Item', got %v", body["title"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var bodyReader *bytes.Reader
			if tt.bodyJSON != "" {
				bodyReader = bytes.NewReader([]byte(tt.bodyJSON))
			} else {
				bodyReader = bytes.NewReader(nil)
			}

			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.url, bodyReader)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			var paramNames []string
			for k, v := range tt.pathParams {
				req.SetPathValue(k, v)
				paramNames = append(paramNames, k)
			}

			ctx, err := core.NewContext(nil, req,
				core.WithPathParams(paramNames),
				core.WithServer(core.Server{
					MaxBodySize: 10 * 1024 * 1024,
				}),
			)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for malformed input, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, ctx)
			}
		})
	}
}
