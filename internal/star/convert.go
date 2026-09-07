package star

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ToValue converts arbitrary Go primitives, slices, and maps into standard Starlark values.
func ToValue(val any) starlark.Value {
	if val == nil {
		return starlark.None
	}

	switch v := val.(type) {
	case starlark.Value:
		return v
	case string:
		return starlark.String(v)
	case bool:
		return starlark.Bool(v)
	case int:
		return starlark.MakeInt(v)
	case int8:
		return starlark.MakeInt(int(v))
	case int16:
		return starlark.MakeInt(int(v))
	case int32:
		return starlark.MakeInt(int(v))
	case int64:
		return starlark.MakeInt64(v)
	case uint:
		return starlark.MakeUint64(uint64(v))
	case uint8:
		return starlark.MakeUint64(uint64(v))
	case uint16:
		return starlark.MakeUint64(uint64(v))
	case uint32:
		return starlark.MakeUint64(uint64(v))
	case uint64:
		return starlark.MakeUint64(v)
	case float32:
		return starlark.Float(float64(v))
	case float64:
		return starlark.Float(v)
	case map[string]any:
		dict := starlark.NewDict(len(v))
		for k, sv := range v {
			_ = dict.SetKey(starlark.String(k), ToValue(sv))
		}
		return dict
	case map[string]string:
		dict := starlark.NewDict(len(v))
		for k, sv := range v {
			_ = dict.SetKey(starlark.String(k), starlark.String(sv))
		}
		return dict
	case []any:
		list := make([]starlark.Value, len(v))
		for i, item := range v {
			list[i] = ToValue(item)
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

// ToGo converts Starlark values back into clean, standard Go types.
func ToGo(val starlark.Value) any {
	if val == nil || val == starlark.None {
		return nil
	}

	switch v := val.(type) {
	case *CaseInsensitiveDict:
		return v.ToMap()
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
			res[i] = ToGo(v.Index(i))
		}
		return res
	case *starlark.Tuple:
		res := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			res[i] = ToGo(v.Index(i))
		}
		return res
	case *starlark.Dict:
		res := make(map[string]any, v.Len())
		for _, item := range v.Items() {
			k := item.Index(0).String()
			if s, ok := item.Index(0).(starlark.String); ok {
				k = s.GoString()
			}
			res[k] = ToGo(item.Index(1))
		}
		return res
	case *starlarkstruct.Struct:
		res := make(map[string]any)
		for _, name := range v.AttrNames() {
			attrVal, err := v.Attr(name)
			if err != nil {
				continue
			}
			res[name] = ToGo(attrVal)
		}
		return res
	default:
		return v.String()
	}
}
