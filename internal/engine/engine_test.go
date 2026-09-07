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
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
)

func newTestEngine(t *testing.T, manifestContent string) *engine.Engine {
	t.Helper()

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "main.hcl")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(engine.Options{
		ConfigPath: tmpDir,
	})
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

	t.Run("Payload exceeding limit is rejected with 413 Problem Details", func(t *testing.T) {
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
    headers {
      field "x-api-key" {
        type     = string
        required = true
        format   = "uuid"
      }
    }
    query {
      field "source" {
        type    = string
        default = "direct"
      }
    }
    body = schema.user_create
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

	t.Run("Returns 422 with invalid_params on constraint breach", func(t *testing.T) {
		t.Parallel()

		body := strings.NewReader(`{
			"email": "invalid-email",
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

		var p problem.Problem
		if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
			t.Fatalf("failed to decode 422 problem details: %v", err)
		}

		if len(p.InvalidParams) != 4 {
			t.Errorf("expected 4 invalid params, got %d: %+v", len(p.InvalidParams), p.InvalidParams)
		}
	})

	t.Run("Normalizes and injects defaults on valid payload", func(t *testing.T) {
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
			t.Errorf("expected default role 'member', got %v", resp["role"])
		}
		if resp["source"] != "direct" {
			t.Errorf("expected default source 'direct', got %v", resp["source"])
		}
	})
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
				Detail:   "Provide a valid API key.",
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
			t.Fatalf("status code = %d; want %d", rec.Code, http.StatusUnauthorized)
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
			t.Fatalf("status code = %d; want 500", rec.Code)
		}
	})
}

func TestEngine_OpenAPIRoutes(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, `
openapi {
  title   = "Store API"
  version = "1.0.0"
}

endpoint "GET /docs" {
  openapi "ui" {
    renderer = "scalar"
  }
}

endpoint "GET /openapi.json" {
  openapi "spec" {
    format = "json"
  }
}

endpoint "GET /openapi.yaml" {
  openapi "spec" {
    format = "yaml"
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
		if !strings.Contains(rec.Body.String(), "scalar") {
			t.Errorf("expected Scalar CDN reference in html response")
		}
	})

	t.Run("Serves raw OpenAPI 3.1 JSON at /openapi.json with ETag caching", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.json", http.NoBody)
		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		etag := rec.Header().Get("ETag")
		if etag == "" {
			t.Fatal("expected ETag header on openapi spec response")
		}

		// Conditional request
		reqCond := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.json", http.NoBody)
		reqCond.Header.Set("If-None-Match", etag)
		recCond := httptest.NewRecorder()
		eng.Handler().ServeHTTP(recCond, reqCond)

		if recCond.Code != http.StatusNotModified {
			t.Errorf("expected status 304, got %d", recCond.Code)
		}
	})
}
