package eval_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/eval"
)

func parseExpr(t *testing.T, src string) hcl.Expression {
	t.Helper()

	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("failed to parse test expression %q: %s", src, diags.Error())
	}

	hclExpr := expr.(hcl.Expression)
	return hclExpr
}

func evalExpr(t *testing.T, src string) (any, error) {
	t.Helper()

	ctx := &core.Context{
		Request: &core.RequestState{},
		Steps:   make(map[string]core.StepResult),
	}

	return eval.Any(parseExpr(t, src), ctx)
}

func TestEval(t *testing.T) {
	t.Parallel()

	sampleCtx := &core.Context{
		Request: &core.RequestState{
			Method: "POST",
			Path:   map[string]string{"id": "usr_99"},
			Headers: map[string]string{
				"authorization": "Bearer token",
			},
			Body: map[string]any{
				"role":  "admin",
				"score": 100,
			},
		},
		Steps: map[string]core.StepResult{
			"lookup": {
				"result": map[string]any{
					"exists": true,
					"tier":   "enterprise",
				},
			},
		},
	}

	t.Run("EvalBool", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			expr       hcl.Expression
			defaultVal bool
			expected   bool
		}{
			{
				name:       "Resolves true context comparison",
				expr:       parseExpr(t, `ctx.request.body.role == "admin"`),
				defaultVal: false,
				expected:   true,
			},
			{
				name:       "Resolves step result comparison",
				expr:       parseExpr(t, `steps.lookup.result.exists == false`),
				defaultVal: true,
				expected:   false,
			},
			{
				name:       "Returns default on nil expression",
				expr:       nil,
				defaultVal: true,
				expected:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				res, err := eval.Bool(tt.expr, sampleCtx, tt.defaultVal)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, res)
				}
			})
		}
	})

	t.Run("EvalInt", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			expr       hcl.Expression
			defaultVal int
			expected   int
		}{
			{
				name:       "Evaluates literal integer",
				expr:       parseExpr(t, "404"),
				defaultVal: 200,
				expected:   404,
			},
			{
				name:       "Evaluates arithmetic expression",
				expr:       parseExpr(t, "ctx.request.body.score + 50"),
				defaultVal: 0,
				expected:   150,
			},
			{
				name:       "Returns fallback on nil expression",
				expr:       nil,
				defaultVal: 200,
				expected:   200,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				res, err := eval.Int(tt.expr, sampleCtx, tt.defaultVal)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, res)
				}
			})
		}
	})

	t.Run("EvalMap", func(t *testing.T) {
		t.Parallel()

		expr := parseExpr(t, `{ user_id = ctx.request.path.id, user_tier = steps.lookup.result.tier }`)
		res, err := eval.Map(expr, sampleCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res["user_id"] != "usr_99" {
			t.Errorf("expected 'usr_99', got %v", res["user_id"])
		}
		if res["user_tier"] != "enterprise" {
			t.Errorf("expected 'enterprise', got %v", res["user_tier"])
		}
	})
}
