// Package eval translates runtime core.Context data into HCL EvalContext structures
// and dynamically evaluates HCL AST expressions back into Go primitives.
package eval

import (
	"fmt"
	"math/big"

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
	reqDict := map[string]cty.Value{
		"method":  cty.StringVal(ctx.Request.Method),
		"path":    mapToCty(ctx.Request.Path),
		"query":   mapToCty(ctx.Request.Query),
		"headers": mapToCty(ctx.Request.Headers),
		"body":    anyToCty(ctx.Request.Body),
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
	}
}

func mapToCty(m map[string]string) cty.Value {
	if len(m) == 0 {
		return cty.EmptyObjectVal
	}

	dict := make(map[string]cty.Value, len(m))
	for k, v := range m {
		dict[k] = cty.StringVal(v)
	}

	return cty.ObjectVal(dict)
}

func anyToCty(val any) cty.Value {
	if val == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}

	switch v := val.(type) {
	case cty.Value:
		return v
	case string:
		return cty.StringVal(v)
	case bool:
		return cty.BoolVal(v)
	case int:
		return cty.NumberIntVal(int64(v))
	case int8:
		return cty.NumberIntVal(int64(v))
	case int16:
		return cty.NumberIntVal(int64(v))
	case int32:
		return cty.NumberIntVal(int64(v))
	case int64:
		return cty.NumberIntVal(v)
	case uint:
		return cty.NumberIntVal(int64(v))
	case uint8:
		return cty.NumberIntVal(int64(v))
	case uint16:
		return cty.NumberIntVal(int64(v))
	case uint32:
		return cty.NumberIntVal(int64(v))
	case uint64:
		return cty.NumberIntVal(int64(v))
	case float32:
		return cty.NumberFloatVal(float64(v))
	case float64:
		return cty.NumberFloatVal(v)
	case map[string]any:
		if len(v) == 0 {
			return cty.EmptyObjectVal
		}
		dict := make(map[string]cty.Value, len(v))
		for key, sub := range v {
			dict[key] = anyToCty(sub)
		}
		return cty.ObjectVal(dict)
	case map[string]string:
		if len(v) == 0 {
			return cty.EmptyObjectVal
		}
		dict := make(map[string]cty.Value, len(v))
		for key, sub := range v {
			dict[key] = cty.StringVal(sub)
		}
		return cty.ObjectVal(dict)
	case []any:
		if len(v) == 0 {
			return cty.EmptyTupleVal
		}
		list := make([]cty.Value, len(v))
		for i, item := range v {
			list[i] = anyToCty(item)
		}
		return cty.TupleVal(list)
	case []string:
		if len(v) == 0 {
			return cty.EmptyTupleVal
		}
		list := make([]cty.Value, len(v))
		for i, item := range v {
			list[i] = cty.StringVal(item)
		}
		return cty.TupleVal(list)
	default:
		return cty.StringVal(fmt.Sprintf("%v", v))
	}
}

func ctyToAny(val cty.Value) any {
	if !val.IsKnown() || val.IsNull() {
		return nil
	}

	ty := val.Type()
	switch {
	case ty.Equals(cty.String):
		return val.AsString()
	case ty.Equals(cty.Number):
		bf := val.AsBigFloat()
		if i, acc := bf.Int64(); acc == big.Exact {
			return i
		}
		f, _ := bf.Float64()
		return f
	case ty.Equals(cty.Bool):
		return val.True()
	case ty.IsObjectType() || ty.IsMapType():
		m := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			m[k.AsString()] = ctyToAny(v)
		}
		return m
	case ty.IsTupleType() || ty.IsListType() || ty.IsSetType():
		var l []any
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			l = append(l, ctyToAny(v))
		}
		return l
	default:
		return nil
	}
}
