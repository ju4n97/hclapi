// Package eval translates runtime core.Context data into HCL EvalContext structures
// and dynamically evaluates HCL AST expressions back into Go primitives.
package eval

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/ju4n97/hclapi/internal/core"
)

// baseEvalContext is allocated once at package load and reused across all requests and parsing.
var baseEvalContext = &hcl.EvalContext{
	Functions: StandardFunctions(),
}

// BaseContext returns the singleton root EvalContext containing all standard functions.
func BaseContext() *hcl.EvalContext {
	return baseEvalContext
}

// Bool evaluates an HCL expression to a boolean value.
func Bool(expr hcl.Expression, ctx *core.Context, defaultVal bool) (bool, error) {
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

// Int evaluates an HCL expression to an integer.
func Int(expr hcl.Expression, ctx *core.Context, defaultVal int) (int, error) {
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

// Map evaluates an HCL expression into a string-keyed dictionary.
func Map(expr hcl.Expression, ctx *core.Context) (map[string]any, error) {
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

// Any evaluates an arbitrary HCL expression into corresponding Go types.
func Any(expr hcl.Expression, ctx *core.Context) (any, error) {
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
	if ctx == nil {
		return baseEvalContext
	}

	var reqVal cty.Value
	if ctx.Request != nil {
		reqVal = cty.ObjectVal(map[string]cty.Value{
			"method":  cty.StringVal(ctx.Request.Method),
			"path":    mapToCty(ctx.Request.Path),
			"query":   mapToCty(ctx.Request.Query),
			"headers": mapToCty(ctx.Request.Headers),
			"body":    anyToCty(ctx.Request.Body),
		})
	} else {
		reqVal = cty.EmptyObjectVal
	}

	stepsDict := make(map[string]cty.Value, len(ctx.Steps))
	for name, res := range ctx.Steps {
		stepsDict[name] = cty.ObjectVal(map[string]cty.Value{
			"result":        anyToCty(res.Result),
			"rows_affected": cty.NumberIntVal(res.RowsAffected),
		})
	}

	childCtx := baseEvalContext.NewChild()

	childCtx.Variables = map[string]cty.Value{
		"ctx": cty.ObjectVal(map[string]cty.Value{
			"request":         reqVal,
			"timestamp_epoch": cty.NumberIntVal(ctx.TimestampEpoch),
		}),
		"steps": cty.ObjectVal(stepsDict),
	}

	return childCtx
}
