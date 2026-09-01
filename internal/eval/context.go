package eval

import (
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/ju4n97/hclapi/internal/core"
)

// buildBaseFunctions combines standard runtime functions with schema type constructors.
func buildBaseFunctions() map[string]function.Function {
	funcs := standardFunctions()
	maps.Copy(funcs, typeConstructors())
	return funcs
}

// baseEvalContext is allocated once at package load and reused across all requests and parsing.
var baseEvalContext = &hcl.EvalContext{
	Variables: map[string]cty.Value{
		"string": cty.StringVal("string"),
		"int":    cty.StringVal("int"),
		"float":  cty.StringVal("float"),
		"bool":   cty.StringVal("bool"),
		"any":    cty.StringVal("any"),
	},
	Functions: buildBaseFunctions(),
}

// BaseContext returns the singleton root EvalContext containing all standard functions.
func BaseContext() *hcl.EvalContext {
	return baseEvalContext
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
	for name, stepExports := range ctx.Steps {
		stepsDict[name] = anyToCty(stepExports)
	}

	childCtx := baseEvalContext.NewChild()

	childCtx.Functions = map[string]function.Function{
		"problem": problemFunc(ctx),
	}

	childCtx.Variables = map[string]cty.Value{
		"ctx": cty.ObjectVal(map[string]cty.Value{
			"request":         reqVal,
			"timestamp_epoch": cty.NumberIntVal(ctx.TimestampEpoch),
		}),
		"steps": cty.ObjectVal(stepsDict),
	}

	return childCtx
}
