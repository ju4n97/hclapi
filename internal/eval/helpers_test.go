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

	return eval.EvalAny(parseExpr(t, src), ctx)
}
