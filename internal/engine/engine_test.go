package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/engine"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
)

// newTestEngine compiles an in-memory HCL manifest into an isolated Engine instance.
func newTestEngine(t *testing.T, m string, opts ...func(*manifest.Options)) *engine.Engine {
	t.Helper()

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.hcl")
	if err := os.WriteFile(manifestPath, []byte(m), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	options := manifest.Options{ConfigPath: tmpDir}
	for _, opt := range opts {
		opt(&options)
	}

	eng, err := engine.New(options)
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	return eng
}

func TestEngine_RoutingAndCatchAll(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, `
endpoint "GET /ping" {
  pipeline {
    respond {
      status = 200
      body   = { status = "pong" }
    }
  }
}

endpoint "GET /users/{id}" {
  pipeline {
    respond {
      status = 200
      body   = { user_id = ctx.request.path.id }
    }
  }
}

endpoint "GET /static/{filepath...}" {
  pipeline {
    respond {
      status = 200
      body   = { file = ctx.request.path.filepath }
    }
  }
}
`)

	t.Run("Static route matching", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("Single path parameter extraction", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/99", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["user_id"] != "99" {
			t.Errorf("expected user_id '99', got %v", body["user_id"])
		}
	})

	t.Run("Catch-all wildcard path extraction", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/static/css/theme/dark.css", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["file"] != "css/theme/dark.css" {
			t.Errorf("expected file 'css/theme/dark.css', got %v", body["file"])
		}
	})
}

func TestEngine_MaxBodySizeEnforcement(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, `
server {
  max_body_size = "1KB"
}

endpoint "POST /upload" {
  pipeline {
    respond {
      status = 200
      body   = { status = "accepted" }
    }
  }
}
`)

	t.Run("Payload under limit succeeds", func(t *testing.T) {
		t.Parallel()

		smallBody := strings.NewReader(`{"name": "tiny payload"}`)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", smallBody)
		rec := httptest.NewRecorder()

		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Payload exceeding 1KB is rejected with 413 Problem Details", func(t *testing.T) {
		t.Parallel()

		largePayload := fmt.Sprintf(`{"data": %q}`, strings.Repeat("A", 2048))
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", strings.NewReader(largePayload))
		rec := httptest.NewRecorder()

		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status 413, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestEngine_SchemaValidationIngress(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, `
schema "pagination" {
  field "source" {
    type    = string
    default = "direct"
  }
}

schema "auth_headers" {
  field "x-api-key" {
    type     = string
    required = true
    format   = "uuid"
  }
}

schema "user_create" {
  field "email" {
    type     = string
    required = true
    format   = "email"
  }
  field "username" {
    type       = string
    required   = true
    min_length = 3
  }
  field "role" {
    type    = string
    default = "member"
    enum    = ["admin", "member"]
  }
}

endpoint "POST /api/v1/users" {
  request {
    headers = schema.auth_headers
    query   = schema.pagination
    body    = schema.user_create
  }

  pipeline {
    respond {
      status = 201
      body = {
        email    = ctx.request.body.email
        username = ctx.request.body.username
        role     = ctx.request.body.role
        source   = ctx.request.query.source
      }
    }
  }
}
`)

	t.Run("Returns 422 with all invalid_params when header, email, and enum are invalid", func(t *testing.T) {
		t.Parallel()

		body := strings.NewReader(`{
			"email": "invalid-email-format",
			"username": "ab",
			"role": "superadmin"
		}`)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/users", body)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var problem problem.Problem
		if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
			t.Fatalf("failed to decode 422 problem details: %v", err)
		}

		if len(problem.InvalidParams) != 4 {
			t.Errorf(
				"expected 4 invalid params (x-api-key, email, username length, role enum), got %d: %+v",
				len(problem.InvalidParams),
				problem.InvalidParams,
			)
		}
	})

	t.Run("Succeeds, injects defaults, and normalizes valid payload", func(t *testing.T) {
		t.Parallel()

		body := strings.NewReader(`{
			"email": "jane@example.com",
			"username": "jane_doe"
		}`)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/users", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "f47ac10b-58cc-4372-a567-0e02b2c3d479")

		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["email"] != "jane@example.com" {
			t.Errorf("expected email 'jane@example.com', got %v", resp["email"])
		}
		if resp["role"] != "member" {
			t.Errorf("expected injected default role 'member', got %v", resp["role"])
		}
		if resp["source"] != "direct" {
			t.Errorf("expected injected query default 'direct', got %v", resp["source"])
		}
	})
}

func TestEngine_HeaderCaseInsensitivity_RFC9110(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, `
endpoint "GET /api/v1/headers" {
  request {
    headers {
      field "Authorization" {
        type     = string
        required = true
      }
      field "X-Api-Key" {
        type     = string
        required = false
      }
    }
  }

  pipeline {
    respond {
      status = 200
      body = {
        auth = ctx.request.headers.authorization
      }
    }
  }
}
`)

	tests := []struct {
		name         string
		headerKey    string
		headerValue  string
		wantStatus   int
		wantErrorKey string
	}{
		{
			name:        "exact casing as declared in schema (Authorization)",
			headerKey:   "Authorization",
			headerValue: "Bearer secret-token",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "all lowercase (authorization)",
			headerKey:   "authorization",
			headerValue: "Bearer secret-token",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "all uppercase (AUTHORIZATION)",
			headerKey:   "AUTHORIZATION",
			headerValue: "Bearer secret-token",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "mixed case (AuThOrIzAtIoN)",
			headerKey:   "AuThOrIzAtIoN",
			headerValue: "Bearer secret-token",
			wantStatus:  http.StatusOK,
		},
		{
			name:         "missing required header",
			headerKey:    "",
			headerValue:  "",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrorKey: "Authorization", // Preserves schema casing in error diagnostic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/headers", http.NoBody)
			if tt.headerKey != "" {
				req.Header.Set(tt.headerKey, tt.headerValue)
			}
			rec := httptest.NewRecorder()

			eng.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status code = %d; want %d. Body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantErrorKey != "" {
				var p problem.Problem
				if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
					t.Fatalf("failed to decode response JSON: %v", err)
				}

				if len(p.InvalidParams) == 0 {
					t.Fatalf("expected InvalidParams, got none")
				}
				if p.InvalidParams[0].Name != tt.wantErrorKey {
					t.Errorf("InvalidParams[0].Name = %q; want %q", p.InvalidParams[0].Name, tt.wantErrorKey)
				}
			}
		})
	}
}

func TestEngine_ProblemError(t *testing.T) {
	t.Parallel()

	manifest := `
endpoint "POST /api/v1/secure" {
  pipeline {
    go "auth_check" {
      use = "auth.verify"
    }
    respond {
      status = 200
      body   = { status = "ok" }
    }
  }
}
`

	t.Run("custom problem.Problem preserves status and fields", func(t *testing.T) {
		t.Parallel()
		eng := newTestEngine(t, manifest)

		err := eng.RegisterStep("auth.verify", func(ctx context.Context, step *runtime.Step) (any, error) {
			return nil, problem.Problem{
				Type:     "urn:hclapi:error:missing-api-key",
				Title:    "Missing API key",
				Status:   http.StatusUnauthorized,
				Detail:   "Provide a valid API key in the 'Authorization' header.",
				Step:     step.Name,
				Instance: "/api/v1/secure",
			}
		})
		if err != nil {
			t.Fatalf("failed to register step: %v", err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/secure", http.NoBody)
		rec := httptest.NewRecorder()

		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status code = %d; want %d. Body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/problem+json" {
			t.Errorf("Content-Type = %q; want application/problem+json", contentType)
		}

		var p problem.Problem
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if p.Status != http.StatusUnauthorized {
			t.Errorf("problem.Status = %d; want %d", p.Status, http.StatusUnauthorized)
		}
		if p.Title != "Missing API key" {
			t.Errorf("problem.Title = %q; want 'Missing API key'", p.Title)
		}
		if p.Type != "urn:hclapi:error:missing-api-key" {
			t.Errorf("problem.Type = %q; want 'urn:hclapi:error:missing-api-key'", p.Type)
		}
		if p.Detail != "Provide a valid API key in the 'Authorization' header." {
			t.Errorf("problem.Detail = %q; want expected detail", p.Detail)
		}
	})

	t.Run("generic error converts to 500 pipeline failure", func(t *testing.T) {
		t.Parallel()
		eng := newTestEngine(t, manifest)

		err := eng.RegisterStep("auth.verify", func(ctx context.Context, step *runtime.Step) (any, error) {
			return nil, errors.New("database connection refused")
		})
		if err != nil {
			t.Fatalf("failed to register step: %v", err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/secure", http.NoBody)
		rec := httptest.NewRecorder()

		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status code = %d; want %d", rec.Code, http.StatusInternalServerError)
		}

		var p problem.Problem
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if p.Status != 500 {
			t.Errorf("problem.Status = %d; want 500", p.Status)
		}
		if p.Type != "urn:hclapi:error:pipeline-execution-failed" {
			t.Errorf("problem.Type = %q; want pipeline-execution-failed URN", p.Type)
		}
	})

	t.Run("step panic is recovered and converts to 500", func(t *testing.T) {
		t.Parallel()
		eng := newTestEngine(t, manifest)

		err := eng.RegisterStep("auth.verify", func(ctx context.Context, step *runtime.Step) (any, error) {
			panic("unexpected memory crash")
		})
		if err != nil {
			t.Fatalf("failed to register step: %v", err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/secure", http.NoBody)
		rec := httptest.NewRecorder()

		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status code = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestEngine_ProblemAutoDerivationAndHelpers(t *testing.T) {
	t.Parallel()

	manifest := `
endpoint "POST /api/v1/checkout" {
  pipeline {
    go "process_payment" {
      use = "payment.charge"
    }
    respond {
      status = 200
      body   = { ok = true }
    }
  }
}
`

	t.Run("step.Problem helper binds step name and derives status/title", func(t *testing.T) {
		t.Parallel()
		eng := newTestEngine(t, manifest)

		err := eng.RegisterStep("payment.charge", func(ctx context.Context, step *runtime.Step) (any, error) {
			return nil, step.Problem(http.StatusPaymentRequired, "Insufficient card balance")
		})
		if err != nil {
			t.Fatalf("failed to register step: %v", err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/checkout", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusPaymentRequired {
			t.Fatalf("status = %d; want 402", rec.Code)
		}

		var p problem.Problem
		_ = json.Unmarshal(rec.Body.Bytes(), &p)

		if p.Step != "process_payment" {
			t.Errorf("p.Step = %q; want 'process_payment'", p.Step)
		}
		if p.Title != "Payment Required" {
			t.Errorf("p.Title = %q; want 'Payment Required'", p.Title)
		}
		if p.Type != "urn:hclapi:error:payment-required" {
			t.Errorf("p.Type = %q; want 'urn:hclapi:error:payment-required'", p.Type)
		}
		if p.Detail != "Insufficient card balance" {
			t.Errorf("p.Detail = %q; want 'Insufficient card balance'", p.Detail)
		}
	})

	t.Run("bare Problem struct auto-derives title, type, and instance", func(t *testing.T) {
		t.Parallel()
		eng := newTestEngine(t, manifest)

		err := eng.RegisterStep("payment.charge", func(ctx context.Context, step *runtime.Step) (any, error) {
			return nil, problem.Problem{
				Status: http.StatusForbidden,
				Detail: "Card brand not supported",
			}
		})
		if err != nil {
			t.Fatalf("failed to register step: %v", err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/checkout", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d; want 403", rec.Code)
		}

		var p problem.Problem
		_ = json.Unmarshal(rec.Body.Bytes(), &p)

		if p.Title != "Forbidden" {
			t.Errorf("auto-derived Title = %q; want 'Forbidden'", p.Title)
		}
		if p.Type != "urn:hclapi:error:forbidden" {
			t.Errorf("auto-derived Type = %q; want 'urn:hclapi:error:forbidden'", p.Type)
		}
		if p.Instance != "/api/v1/checkout" {
			t.Errorf("auto-derived Instance = %q; want '/api/v1/checkout'", p.Instance)
		}
	})

	t.Run("Problem with custom extensions flattens into root JSON payload", func(t *testing.T) {
		t.Parallel()
		eng := newTestEngine(t, manifest)

		err := eng.RegisterStep("payment.charge", func(ctx context.Context, step *runtime.Step) (any, error) {
			p := step.Problem(http.StatusTooManyRequests, "Rate limit reached")
			p.Extensions = map[string]any{
				"retry_after_ms": 5000,
				"error_code":     "RATE_LIMIT_EXCEEDED",
			}
			return nil, p
		})
		if err != nil {
			t.Fatalf("failed to register step: %v", err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/checkout", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		var rawMap map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &rawMap)

		if rawMap["error_code"] != "RATE_LIMIT_EXCEEDED" {
			t.Errorf("expected root error_code, got %v", rawMap["error_code"])
		}
		if rawMap["retry_after_ms"] != float64(5000) {
			t.Errorf("expected root retry_after_ms, got %v", rawMap["retry_after_ms"])
		}
	})
}

func TestEngine_OpenAPIRoutes(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, `
server {
  openapi {
    title   = "Store API"
    version = "1.0.0"
  }
}

endpoint "GET /docs" {
  openapi {
    ui = "scalar"
  }
}

endpoint "GET /openapi.json" {
  openapi {
    format = "json"
  }
}

endpoint "GET /openapi.yaml" {
  openapi {
    format = "yaml"
  }
}

endpoint "GET /ping" {
  description = "Health check"
  pipeline {
    respond {
      status = 200
      body   = { ok = true }
    }
  }
}
`)

	t.Run("Serves interactive Scalar documentation at /docs", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/docs", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("expected text/html, got %q", ct)
		}
		if !strings.Contains(rec.Body.String(), "scalar") {
			t.Errorf("expected Scalar CDN reference in html response")
		}
	})

	t.Run("Serves raw OpenAPI 3.1 JSON at /openapi.json", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.json", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("expected application/json, got %q", ct)
		}

		var doc map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&doc); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}
		if doc["openapi"] != "3.1.0" {
			t.Errorf("expected openapi 3.1.0, got %v", doc["openapi"])
		}
	})

	t.Run("Serves raw OpenAPI 3.1 YAML at /openapi.yaml", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.yaml", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/yaml") {
			t.Errorf("expected application/yaml, got %q", ct)
		}
		if !strings.Contains(rec.Body.String(), "openapi: 3.1.0") {
			t.Errorf("expected yaml header in response")
		}
	})
}
