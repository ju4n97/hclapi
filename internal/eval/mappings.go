package eval

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/zclconf/go-cty/cty"
)

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
	case []map[string]any:
		if len(v) == 0 {
			return cty.EmptyTupleVal
		}
		list := make([]cty.Value, len(v))
		for i, item := range v {
			list[i] = anyToCty(item)
		}
		return cty.TupleVal(list)
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
	}

	// Reflection fallback for any other slice, array, or custom map
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Len() == 0 {
			return cty.EmptyTupleVal
		}
		list := make([]cty.Value, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			list[i] = anyToCty(rv.Index(i).Interface())
		}
		return cty.TupleVal(list)
	case reflect.Map:
		if rv.Len() == 0 {
			return cty.EmptyObjectVal
		}
		dict := make(map[string]cty.Value, rv.Len())
		for _, key := range rv.MapKeys() {
			dict[fmt.Sprintf("%v", key.Interface())] = anyToCty(rv.MapIndex(key).Interface())
		}
		return cty.ObjectVal(dict)
	default:
		return cty.StringVal(fmt.Sprintf("%v", val))
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
		if l == nil {
			return []any{}
		}
		return l
	default:
		return nil
	}
}
