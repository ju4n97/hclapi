package xstarlark

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// Eval compiles and executes a Starlark script against a Go context data map.
func Eval(source string, ctxData map[string]any) (any, error) {
	thread := &starlark.Thread{Name: "hclapi-starlark-vm"}

	opts := &syntax.FileOptions{}

	globals, err := starlark.ExecFileOptions(opts, thread, "manifest.star", source, nil)
	if err != nil {
		return nil, fmt.Errorf("starlark script compilation error: %w", err)
	}

	execVal, exists := globals["execute"]
	if !exists {
		return nil, errors.New("starlark script must define an 'execute(ctx)' function")
	}

	execCallable, ok := execVal.(starlark.Callable)
	if !ok {
		return nil, errors.New("'execute' must be a callable function")
	}

	starlarkCtx := GoToStarlarkValue(ctxData)

	resultVal, err := starlark.Call(thread, execCallable, starlark.Tuple{starlarkCtx}, nil)
	if err != nil {
		return nil, fmt.Errorf("starlark runtime error: %w", err)
	}

	return StarlarkToGoValue(resultVal), nil
}

// GoToStarlarkValue converts arbitrary Go data into Starlark values.
func GoToStarlarkValue(val any) starlark.Value {
	if val == nil {
		return starlark.None
	}

	switch v := val.(type) {
	case string:
		return starlark.String(v)
	case bool:
		return starlark.Bool(v)
	case int:
		return starlark.MakeInt(v)
	case int64:
		return starlark.MakeInt64(v)
	case float64:
		return starlark.Float(v)
	case map[string]any:
		dict := make(starlark.StringDict)
		for key, subVal := range v {
			dict[key] = GoToStarlarkValue(subVal)
		}
		return starlarkstruct.FromStringDict(starlarkstruct.Default, dict)
	case map[string]string:
		dict := make(starlark.StringDict)
		for key, subVal := range v {
			dict[key] = starlark.String(subVal)
		}
		return starlarkstruct.FromStringDict(starlarkstruct.Default, dict)
	case []any:
		var list []starlark.Value
		for _, item := range v {
			list = append(list, GoToStarlarkValue(item))
		}
		return starlark.NewList(list)
	case []string:
		var list []starlark.Value
		for _, item := range v {
			list = append(list, starlark.String(item))
		}
		return starlark.NewList(list)
	default:
		return starlark.String(fmt.Sprintf("%v", v))
	}
}

// StarlarkToGoValue converts Starlark values back into standard Go structures.
func StarlarkToGoValue(val starlark.Value) any {
	if val == nil || val == starlark.None {
		return nil
	}

	switch v := val.(type) {
	case starlark.String:
		return v.GoString()
	case starlark.Bool:
		return bool(v)
	case starlark.Int:
		i, _ := v.Int64()
		return i
	case starlark.Float:
		return float64(v)
	case *starlark.List:
		var res []any
		for i := 0; i < v.Len(); i++ {
			res = append(res, StarlarkToGoValue(v.Index(i)))
		}
		return res
	case *starlark.Dict:
		res := make(map[string]any)
		for _, item := range v.Items() {
			k := item.Index(0).String()
			if s, ok := item.Index(0).(starlark.String); ok {
				k = s.GoString()
			}
			res[k] = StarlarkToGoValue(item.Index(1))
		}
		return res
	case *starlarkstruct.Struct:
		res := make(map[string]any)
		for _, name := range v.AttrNames() {
			val, _ := v.Attr(name)
			res[name] = StarlarkToGoValue(val)
		}
		return res
	default:
		return v.String()
	}
}
