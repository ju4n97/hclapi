package star

import (
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// DefaultMaxSteps is the instruction budget per script execution (100k instructions).
const DefaultMaxSteps = 100_000

// Request models incoming HTTP request state for Starlark scripts.
type Request struct {
	Method  string
	Path    map[string]string
	Query   map[string]string
	Headers map[string]string
	Body    any
}

// Env represents the execution context envelope (`ctx`) exposed to Starlark.
type Env struct {
	Request        Request
	Steps          map[string]map[string]any
	TimestampEpoch int64
}

// Eval executes a Starlark script against the environment using the default step budget.
func Eval(source string, env Env) (any, error) {
	return EvalWithLimit(source, env, DefaultMaxSteps)
}

// EvalWithLimit compiles and runs a Starlark script enforcing a strict instruction limit.
func EvalWithLimit(source string, env Env, maxSteps uint64) (any, error) {
	thread := &starlark.Thread{Name: "hclapi-star-vm"}
	if maxSteps > 0 {
		thread.SetMaxExecutionSteps(maxSteps)
	}

	opts := &syntax.FileOptions{
		Set:       true,
		While:     true,
		Recursion: true,
	}

	globals, err := starlark.ExecFileOptions(opts, thread, "manifest.star", source, nil)
	if err != nil {
		return nil, &CompileError{Err: err}
	}

	execVal, exists := globals["execute"]
	if !exists {
		return nil, ErrMissingExecute
	}

	execCallable, ok := execVal.(starlark.Callable)
	if !ok {
		return nil, ErrMissingExecute
	}

	// Construct `ctx.request`
	reqFields := starlark.StringDict{
		"method":  starlark.String(env.Request.Method),
		"path":    ToValue(env.Request.Path),
		"query":   ToValue(env.Request.Query),
		"headers": NewCaseInsensitiveDictFromStrings(env.Request.Headers),
		"body":    ToValue(env.Request.Body),
	}

	// Construct `ctx.steps`
	stepFields := make(starlark.StringDict, len(env.Steps))
	for name, stepExports := range env.Steps {
		stepFields[name] = ToValue(stepExports)
	}

	// Construct `ctx` envelope
	ctxStruct := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"request":         starlarkstruct.FromStringDict(starlarkstruct.Default, reqFields),
		"steps":           starlarkstruct.FromStringDict(starlarkstruct.Default, stepFields),
		"timestamp_epoch": starlark.MakeInt64(env.TimestampEpoch),
	})

	resultVal, err := starlark.Call(thread, execCallable, starlark.Tuple{ctxStruct}, nil)
	if err != nil {
		return nil, &RuntimeError{Err: err}
	}

	return ToGo(resultVal), nil
}
