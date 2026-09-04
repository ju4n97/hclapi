package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/engine"
)

func TestEngine_RoutingAndCatchAll(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manifestContent := `
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
`
	if err := os.WriteFile(filepath.Join(tmpDir, "routes.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{ConfigPath: tmpDir})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

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

	tmpDir := t.TempDir()
	manifestContent := `
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
`
	if err := os.WriteFile(filepath.Join(tmpDir, "server.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{ConfigPath: tmpDir})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

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

	tmpDir := t.TempDir()
	manifestContent := `
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
`
	if err := os.WriteFile(filepath.Join(tmpDir, "routes.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{ConfigPath: tmpDir})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	t.Run("Returns 422 with all invalid_params when header, email, and enum are invalid", func(t *testing.T) {
		t.Parallel()

		body := strings.NewReader(`{
			"email": "invalid-email-format",
			"username": "ab",
			"role": "superadmin"
		}`)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/users", body)
		req.Header.Set("Content-Type", "application/json")
		// x-api-key is intentionally omitted

		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var problem core.ProblemDetailsError
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

func TestEngine_ProblemDetailsError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manifestContent := `
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

	if err := os.WriteFile(filepath.Join(tmpDir, "routes.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{ConfigPath: tmpDir})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	t.Run("custom ProblemDetailsError preserves status and fields", func(t *testing.T) {
		t.Parallel()

		_ = eng.RegisterStep("auth.verify", func(ctx context.Context, step *core.Step) (any, error) {
			return nil, core.ProblemDetailsError{
				Type:     "urn:hclapi:error:missing-api-key",
				Title:    "Missing API key",
				Status:   http.StatusUnauthorized,
				Detail:   "Provide a valid API key in the 'Authorization' header.",
				Step:     "auth_check",
				Instance: "/api/v1/secure",
			}
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/secure", nil)
		rec := httptest.NewRecorder()

		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status code = %d; want %d", rec.Code, http.StatusUnauthorized)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/problem+json" {
			t.Errorf("Content-Type = %q; want application/problem+json", contentType)
		}

		var prob core.ProblemDetailsError
		if err := json.Unmarshal(rec.Body.Bytes(), &prob); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if prob.Status != http.StatusUnauthorized {
			t.Errorf("problem.Status = %d; want %d", prob.Status, http.StatusUnauthorized)
		}
		if prob.Title != "Missing API key" {
			t.Errorf("problem.Title = %q; want 'Missing API key'", prob.Title)
		}
		if prob.Type != "urn:hclapi:error:missing-api-key" {
			t.Errorf("problem.Type = %q; want 'urn:hclapi:error:missing-api-key'", prob.Type)
		}
		if prob.Detail != "Provide a valid API key in the 'Authorization' header." {
			t.Errorf("problem.Detail = %q; want expected detail", prob.Detail)
		}
	})
}

func TestEngine_OpenAPIRoutes(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manifestContent := `
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
`
	if err := os.WriteFile(filepath.Join(tmpDir, "routes.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{ConfigPath: tmpDir})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

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

	t.Run("Serves raw OpenAPI 3.1 JSON at /openapi.json [1.2, 1.3]", func(t *testing.T) {
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
