package xstarlark

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// Eval compiles and executes a Starlark script against a context value.
func Eval(source string, ctxValue starlark.Value) (any, error) {
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

	resultVal, err := starlark.Call(thread, execCallable, starlark.Tuple{ctxValue}, nil)
	if err != nil {
		return nil, fmt.Errorf("starlark runtime error: %w", err)
	}

	return StarlarkToGoValue(resultVal), nil
}

// GoToStarlarkValue converts Go primitives, slices, and maps into standard Starlark values.
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
		dict := starlark.NewDict(len(v))
		for key, subVal := range v {
			_ = dict.SetKey(starlark.String(key), GoToStarlarkValue(subVal))
		}
		return dict
	case map[string]string:
		dict := starlark.NewDict(len(v))
		for key, subVal := range v {
			_ = dict.SetKey(starlark.String(key), starlark.String(subVal))
		}
		return dict
	case []any:
		list := make([]starlark.Value, len(v))
		for i, item := range v {
			list[i] = GoToStarlarkValue(item)
		}
		return starlark.NewList(list)
	case []string:
		list := make([]starlark.Value, len(v))
		for i, item := range v {
			list[i] = starlark.String(item)
		}
		return starlark.NewList(list)
	default:
		return starlark.String(fmt.Sprintf("%v", v))
	}
}

// StarlarkToGoValue converts standard Starlark values back into standard Go structures.
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
		res := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			res[i] = StarlarkToGoValue(v.Index(i))
		}
		return res
	case *starlark.Dict:
		res := make(map[string]any, v.Len())
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
			attrVal, _ := v.Attr(name)
			res[name] = StarlarkToGoValue(attrVal)
		}
		return res
	default:
		return v.String()
	}
}
