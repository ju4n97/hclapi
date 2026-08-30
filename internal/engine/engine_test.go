package engine_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
