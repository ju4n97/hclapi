// Package eval translates runtime core.Context data into HCL EvalContext structures
// and dynamically evaluates HCL AST expressions back into Go primitives.
package eval

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/ju4n97/hclapi/internal/core"
)

// Bool evaluates an HCL expression to a boolean value.
func Bool(expr hcl.Expression, execCtx *core.ExecutionContext, defaultVal bool) (bool, error) {
	if expr == nil {
		return defaultVal, nil
	}

	val, diags := expr.Value(buildEvalContext(execCtx))
	if diags.HasErrors() {
		return false, fmt.Errorf("evaluating boolean expression: %s", diags.Error())
	}

	if !val.IsKnown() || val.IsNull() || val.Type() != cty.Bool {
		return defaultVal, nil
	}

	return val.True(), nil
}

// Int evaluates an HCL expression to an integer.
func Int(expr hcl.Expression, execCtx *core.ExecutionContext, defaultVal int) (int, error) {
	if expr == nil {
		return defaultVal, nil
	}

	val, diags := expr.Value(buildEvalContext(execCtx))
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
func Map(expr hcl.Expression, execCtx *core.ExecutionContext) (map[string]any, error) {
	if expr == nil {
		return nil, nil
	}

	val, diags := expr.Value(buildEvalContext(execCtx))
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
func Any(expr hcl.Expression, execCtx *core.ExecutionContext) (any, error) {
	if expr == nil {
		return nil, nil
	}

	val, diags := expr.Value(buildEvalContext(execCtx))
	if diags.HasErrors() {
		return nil, fmt.Errorf("evaluating expression: %s", diags.Error())
	}

	return ctyToAny(val), nil
}
