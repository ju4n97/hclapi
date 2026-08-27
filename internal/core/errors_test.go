package core_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
)

func TestProblemDetails(t *testing.T) {
	t.Parallel()

	t.Run("DefaultErrorHandler writes application/problem+json", func(t *testing.T) {
		t.Parallel()

		problem := core.ProblemDetails{
			Type:     "https://github.com/ju4n97/hclapi/errors/bad-request",
			Title:    "Invalid JSON Payload",
			Status:   http.StatusBadRequest,
			Detail:   "Syntax error at line 1",
			Instance: "/api/v1/sanitize",
			InvalidParams: []core.InvalidParam{
				{Name: "tags", Reason: "must be a list"},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sanitize", nil)
		rec := httptest.NewRecorder()

		core.DefaultErrorHandler(rec, req, problem)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Errorf("expected Content-Type application/problem+json, got %q", ct)
		}

		var parsed core.ProblemDetails
		if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if parsed.Title != "Invalid JSON Payload" || len(parsed.InvalidParams) != 1 {
			t.Errorf("unexpected problem details structure: %+v", parsed)
		}
	})

	t.Run("Error formatting", func(t *testing.T) {
		t.Parallel()

		prob := core.ProblemDetails{
			Title:  "Bad Request",
			Detail: "missing param",
		}
		if !strings.Contains(prob.Error(), "Bad Request: missing param") {
			t.Errorf("unexpected error string: %s", prob.Error())
		}
	})
}
