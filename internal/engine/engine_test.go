package engine_test

import (
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

func TestCatchAllPathParameters(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manifestContent := `
endpoint "GET /static/{filepath...}" {
  pipeline {
    respond {
      status = 200
      body = {
        path = ctx.request.path.filepath
      }
    }
  }
}

endpoint "GET /orgs/{org_id}/files/{filepath...}" {
  pipeline {
    respond {
      status = 200
      body = {
        org  = ctx.request.path.org_id
        file = ctx.request.path.filepath
      }
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "routes.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{
		ConfigPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	tests := []struct {
		name         string
		requestURL   string
		expectStatus int
		validate     func(t *testing.T, body map[string]any)
	}{
		{
			name:         "Single-level catch-all",
			requestURL:   "/static/favicon.ico",
			expectStatus: http.StatusOK,
			validate: func(t *testing.T, body map[string]any) {
				if body["path"] != "favicon.ico" {
					t.Errorf("expected path 'favicon.ico', got %v", body["path"])
				}
			},
		},
		{
			name:         "Deeply nested multi-segment catch-all",
			requestURL:   "/static/assets/css/theme/dark.min.css",
			expectStatus: http.StatusOK,
			validate: func(t *testing.T, body map[string]any) {
				if body["path"] != "assets/css/theme/dark.min.css" {
					t.Errorf("expected path 'assets/css/theme/dark.min.css', got %v", body["path"])
				}
			},
		},
		{
			name:         "Combined regular path parameter and catch-all",
			requestURL:   "/orgs/acme-corp/files/documents/2026/report.pdf",
			expectStatus: http.StatusOK,
			validate: func(t *testing.T, body map[string]any) {
				if body["org"] != "acme-corp" {
					t.Errorf("expected org 'acme-corp', got %v", body["org"])
				}
				if body["file"] != "documents/2026/report.pdf" {
					t.Errorf("expected file 'documents/2026/report.pdf', got %v", body["file"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.requestURL, nil)
			rec := httptest.NewRecorder()

			eng.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.expectStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectStatus, rec.Code, rec.Body.String())
			}

			var body map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, body)
			}
		})
	}
}

func TestMaxBodySizeEnforcement(t *testing.T) {
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
      body = { status = "accepted" }
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "server.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{
		ConfigPath: tmpDir,
	})
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
			t.Errorf("expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Payload exceeding 1KB is rejected with 413", func(t *testing.T) {
		t.Parallel()

		largePayload := fmt.Sprintf(`{"data": %q}`, strings.Repeat("A", 2048))
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", strings.NewReader(largePayload))
		rec := httptest.NewRecorder()

		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status 413, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		if !strings.Contains(rec.Body.String(), "Request Entity Too Large") {
			t.Errorf("expected problem details payload too large message, got: %s", rec.Body.String())
		}
	})
}

func TestSchemaValidationIngress(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manifestContent := `
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
`
	if err := os.WriteFile(filepath.Join(tmpDir, "routes.hcl"), []byte(manifestContent), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	eng, err := engine.New(core.Options{
		ConfigPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	t.Run("Returns 422 when required header, bad email, and invalid enum are sent", func(t *testing.T) {
		t.Parallel()

		body := strings.NewReader(`{
			"email": "not-an-email",
			"username": "ab",
			"role": "superadmin"
		}`)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/users", body)
		req.Header.Set("Content-Type", "application/json")
		// x-api-key header intentionally omitted

		rec := httptest.NewRecorder()
		eng.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var problem core.ProblemDetailsError
		if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
			t.Fatalf("failed to decode 422 problem details: %v", err)
		}

		if problem.Title != "Unprocessable Entity" {
			t.Errorf("expected title 'Unprocessable Entity', got %q", problem.Title)
		}
		if len(problem.InvalidParams) < 3 {
			t.Errorf(
				"expected at least 3 invalid params (header, email, role), got %d: %+v",
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
		// Injected schema default
		if resp["role"] != "member" {
			t.Errorf("expected injected default role 'member', got %v", resp["role"])
		}
		// Injected query default
		if resp["source"] != "direct" {
			t.Errorf("expected injected query default 'direct', got %v", resp["source"])
		}
	})
}
