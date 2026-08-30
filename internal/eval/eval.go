// Package eval translates runtime core.Context data into HCL EvalContext structures
// and dynamically evaluates HCL AST expressions back into Go primitives.
package eval

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/zclconf/go-cty/cty"
)

// EvalBool evaluates an HCL expression to a boolean value.
func EvalBool(expr hcl.Expression, ctx *core.Context, defaultVal bool) (bool, error) {
	if expr == nil {
		return defaultVal, nil
	}

	val, diags := expr.Value(buildEvalContext(ctx))
	if diags.HasErrors() {
		return false, fmt.Errorf("evaluating boolean expression: %s", diags.Error())
	}

	if !val.IsKnown() || val.IsNull() || val.Type() != cty.Bool {
		return defaultVal, nil
	}

	return val.True(), nil
}

// EvalInt evaluates an HCL expression to an integer.
func EvalInt(expr hcl.Expression, ctx *core.Context, defaultVal int) (int, error) {
	if expr == nil {
		return defaultVal, nil
	}

	val, diags := expr.Value(buildEvalContext(ctx))
	if diags.HasErrors() {
		return defaultVal, fmt.Errorf("evaluating integer expression: %s", diags.Error())
	}

	if !val.IsKnown() || val.IsNull() || val.Type() != cty.Number {
		return defaultVal, nil
	}

	bf := val.AsBigFloat()
	i, _ := bf.Int64()
	return int(i), nil
}

// EvalMap evaluates an HCL expression into a string-keyed dictionary.
func EvalMap(expr hcl.Expression, ctx *core.Context) (map[string]any, error) {
	if expr == nil {
		return nil, nil
	}

	val, diags := expr.Value(buildEvalContext(ctx))
	if diags.HasErrors() {
		return nil, fmt.Errorf("evaluating map expression: %s", diags.Error())
	}

	res := ctyToAny(val)
	if m, ok := res.(map[string]any); ok {
		return m, nil
	}

	return nil, nil
}

// EvalAny evaluates an arbitrary HCL expression into corresponding Go types.
func EvalAny(expr hcl.Expression, ctx *core.Context) (any, error) {
	if expr == nil {
		return nil, nil
	}

	val, diags := expr.Value(buildEvalContext(ctx))
	if diags.HasErrors() {
		return nil, fmt.Errorf("evaluating expression: %s", diags.Error())
	}

	return ctyToAny(val), nil
}

func buildEvalContext(ctx *core.Context) *hcl.EvalContext {
	var reqDict map[string]cty.Value
	if ctx.Request != nil {
		reqDict = map[string]cty.Value{
			"method":  cty.StringVal(ctx.Request.Method),
			"path":    mapToCty(ctx.Request.Path),
			"query":   mapToCty(ctx.Request.Query),
			"headers": mapToCty(ctx.Request.Headers),
			"body":    anyToCty(ctx.Request.Body),
		}
	}

	stepsDict := make(map[string]cty.Value, len(ctx.Steps))
	for name, res := range ctx.Steps {
		stepsDict[name] = cty.ObjectVal(map[string]cty.Value{
			"result":        anyToCty(res.Result),
			"rows_affected": cty.NumberIntVal(res.RowsAffected),
		})
	}

	ctxDict := map[string]cty.Value{
		"request":         cty.ObjectVal(reqDict),
		"timestamp_epoch": cty.NumberIntVal(ctx.TimestampEpoch),
	}

	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"ctx":   cty.ObjectVal(ctxDict),
			"steps": cty.ObjectVal(stepsDict),
		},
		Functions: standardFunctions(),
	}
}
