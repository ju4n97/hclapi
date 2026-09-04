package eval_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/runtime"
)

func TestContext(t *testing.T) {
	t.Parallel()

	t.Run("BaseContext exposes standard type keywords and type constructors", func(t *testing.T) {
		t.Parallel()

		baseCtx := eval.BaseContext()
		if baseCtx == nil {
			t.Fatal("expected BaseContext to not be nil")
		}

		// Verify primitive type variables
		primitives := []string{"string", "int", "float", "bool", "any"}
		for _, p := range primitives {
			val, exists := baseCtx.Variables[p]
			if !exists || val.AsString() != p {
				t.Errorf("expected primitive type variable %q in BaseContext", p)
			}
		}

		// Verify type constructors in Functions
		constructors := []string{"list", "map"}
		for _, c := range constructors {
			if _, exists := baseCtx.Functions[c]; !exists {
				t.Errorf("expected type constructor function %q in BaseContext", c)
			}
		}
	})

	t.Run("Evaluates list and map type expressions cleanly", func(t *testing.T) {
		t.Parallel()

		resList, err := eval.Any(parseExpr(t, `list(string)`), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resList != "list(string)" {
			t.Errorf("expected 'list(string)', got %v", resList)
		}

		resMap, err := eval.Any(parseExpr(t, `map(int)`), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resMap != "map(int)" {
			t.Errorf("expected 'map(int)', got %v", resMap)
		}
	})

	t.Run("Child context scopes request data and steps", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/42", http.NoBody)
		req.Header.Set("Authorization", "Bearer token-123")

		ctx, err := runtime.NewExecutionContext(nil, req, runtime.WithPathParams([]string{"id"}))
		if err != nil {
			t.Fatalf("failed to create core context: %v", err)
		}
		ctx.Steps["fetch"] = map[string]any{"row": map[string]any{"name": "Jane"}}

		// Verify child context reads request and step variables
		resMethod, err := eval.Any(parseExpr(t, `ctx.request.method`), ctx)
		if err != nil || resMethod != "GET" {
			t.Errorf("expected GET, got %v (err: %v)", resMethod, err)
		}

		resStep, err := eval.Any(parseExpr(t, `steps.fetch.row.name`), ctx)
		if err != nil || resStep != "Jane" {
			t.Errorf("expected Jane, got %v (err: %v)", resStep, err)
		}
	})
}
