package hclapi_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ju4n97/hclapi"
)

func TestEngineFacade(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manifestContent := `
endpoint "GET /api/v1/ping" {
  pipeline {
    go "enrich" {
      use = "test.ping"
    }
    respond {
      status = 200
      body   = steps.enrich.result
    }
  }
}
`
	manifestPath := filepath.Join(tmpDir, "Hclapifile")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0600); err != nil {
		t.Fatalf("failed to write manifest fixture: %v", err)
	}

	engine, err := hclapi.NewEngine(hclapi.Options{
		ManifestDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	err = engine.RegisterStep("test.ping", func(ctx *hclapi.Context) (any, error) {
		return map[string]string{"status": "pong"}, nil
	})
	if err != nil {
		t.Fatalf("failed to register step: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()

	engine.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	expectedJSON := `{"status":"pong"}` + "\n"
	if rec.Body.String() != expectedJSON {
		t.Errorf("expected body %q, got %q", expectedJSON, rec.Body.String())
	}
}
