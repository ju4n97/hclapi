package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/engine"
	"github.com/ju4n97/hclapi/internal/parser"
)

func parseExpr(t *testing.T, src string) hcl.Expression {
	t.Helper()
	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("syntax error: %s", diags.Error())
	}
	return expr
}

func TestPipelineExecutor(t *testing.T) {
	t.Parallel()

	t.Run("Executes Go step, Starlark step, and resolves conditional Respond", func(t *testing.T) {
		t.Parallel()

		goSteps := map[string]core.StepHandler{
			"auth.verify": func(ctx *core.Context, args map[string]any) (any, error) {
				return map[string]any{
					"valid": args["token"] == "secret-token",
					"uid":   42,
				}, nil
			},
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "auth",
				Go: &parser.GoStepBlock{
					Use:  "auth.verify",
					Args: parseExpr(t, `{ token = ctx.request.headers.authorization }`),
				},
			},
			{
				Type: parser.StepTypeStarlark,
				Name: "format",
				Starlark: &parser.StarlarkStepBlock{
					Source: `
def execute(ctx):
    auth_data = ctx.steps.auth.get("result", {})
    return {
        "user_id": auth_data.get("uid"),
        "authorized": auth_data.get("valid")
    }
`,
				},
			},
			// Fallback 401 response when unauthorized
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Condition: parseExpr(t, `steps.format.result.authorized == false`),
					Status:    parseExpr(t, `401`),
					Body:      parseExpr(t, `{ error = "unauthorized" }`),
				},
			},
			// Success 200 response when authorized
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Condition: parseExpr(t, `steps.format.result.authorized == true`),
					Status:    parseExpr(t, `200`),
					Body:      parseExpr(t, `steps.format.result`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, goSteps)

		// 1. Test authorized request
		reqAuth := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		reqAuth.Header.Set("Authorization", "secret-token")

		ctxAuth, err := core.NewContext(nil, reqAuth)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		recAuth := httptest.NewRecorder()
		if err := executor.Execute(recAuth, ctxAuth); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if recAuth.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", recAuth.Code)
		}

		var bodyAuth map[string]any
		_ = json.NewDecoder(recAuth.Body).Decode(&bodyAuth)
		if bodyAuth["authorized"] != true || bodyAuth["user_id"] != float64(42) {
			t.Errorf("unexpected body payload: %+v", bodyAuth)
		}

		// 2. Test unauthorized request (hits 401 condition)
		reqUnauth := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		reqUnauth.Header.Set("Authorization", "wrong-token")

		ctxUnauth, err := core.NewContext(nil, reqUnauth)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		recUnauth := httptest.NewRecorder()
		if err := executor.Execute(recUnauth, ctxUnauth); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if recUnauth.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", recUnauth.Code)
		}

		if !strings.Contains(recUnauth.Body.String(), "unauthorized") {
			t.Errorf("expected unauthorized error body, got: %s", recUnauth.Body.String())
		}
	})

	t.Run("Context cancellation aborts pipeline execution before subsequent steps", func(t *testing.T) {
		t.Parallel()

		stepTwoExecuted := false
		goSteps := map[string]core.StepHandler{
			"step.one": func(ctx *core.Context, args map[string]any) (any, error) {
				return "done_one", nil
			},
			"step.two": func(ctx *core.Context, args map[string]any) (any, error) {
				stepTwoExecuted = true
				return "done_two", nil
			},
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "first",
				Go:   &parser.GoStepBlock{Use: "step.one"},
			},
			{
				Type: parser.StepTypeGo,
				Name: "second",
				Go:   &parser.GoStepBlock{Use: "step.two"},
			},
		}

		executor := engine.NewPipelineExecutor(steps, goSteps)

		reqCtx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel before execution

		req := httptest.NewRequestWithContext(reqCtx, http.MethodGet, "/test", http.NoBody)

		hclapiCtx, err := core.NewContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		err = executor.Execute(rec, hclapiCtx)

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got: %v", err)
		}

		if stepTwoExecuted {
			t.Errorf("expected step two to be skipped after context cancellation")
		}
	})

	t.Run("Omitted body produces an empty response", func(t *testing.T) {
		t.Parallel()

		goSteps := map[string]core.StepHandler{
			"mock.step": func(ctx *core.Context, args map[string]any) (any, error) {
				return "should_be_ignored", nil
			},
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "do_work",
				Go:   &parser.GoStepBlock{Use: "mock.step"},
			},
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Status: parseExpr(t, `204`),
					// Body is intentionally omitted
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, goSteps)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		hclapiCtx, err := core.NewContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, hclapiCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}

		if rec.Body.Len() != 0 {
			t.Errorf("expected empty body, got %q", rec.Body.String())
		}
	})

	t.Run("Evaluates custom headers in respond step", func(t *testing.T) {
		t.Parallel()

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Status: parseExpr(t, `200`),
					Headers: parseExpr(t, `{
						"X-Custom-Header" = "custom_value"
						"X-Echo-Method"   = ctx.request.method
					}`),
					Body: parseExpr(t, `{ ok = true }`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		hclapiCtx, err := core.NewContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, hclapiCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if h := rec.Header().Get("X-Custom-Header"); h != "custom_value" {
			t.Errorf("expected header 'custom_value', got %q", h)
		}

		if h := rec.Header().Get("X-Echo-Method"); h != "GET" {
			t.Errorf("expected header 'GET', got %q", h)
		}
	})
}
