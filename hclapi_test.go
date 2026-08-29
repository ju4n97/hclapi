package hclapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi"
)

func TestCustomErrorHandler(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manifestContent := `
endpoint "POST /api/v1/data" {
  pipeline {
    respond {
      status = 200
      body   = { ok = true }
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.hcl"), []byte(manifestContent), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	customHandlerCalled := false
	customHandler := func(w http.ResponseWriter, r *http.Request, problem hclapi.ProblemDetails) {
		customHandlerCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(problem.Status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"custom_error": problem.Detail,
		})
	}

	engine, err := hclapi.NewEngine(hclapi.Options{
		ConfigPath:   tmpDir,
		ErrorHandler: customHandler,
	})
	if err != nil {
		t.Fatalf("failed to initialize engine: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", strings.NewReader(`{"bad": json`)) // malformed JSON payload
	rec := httptest.NewRecorder()

	engine.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	if !customHandlerCalled {
		t.Errorf("expected custom error handler to be invoked")
	}

	if !strings.Contains(rec.Body.String(), "custom_error") {
		t.Errorf("expected custom error response, got: %s", rec.Body.String())
	}
}
