package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/ju4n97/hclapi/internal/scalar"
)

// Args represents evaluated arguments passed to a Go step from an HCL manifest.
type Args map[string]any

// Has reports whether key exists in the arguments map and is not nil.
func (a Args) Has(key string) bool {
	if a == nil {
		return false
	}
	val, ok := a[key]
	return ok && val != nil
}

// Get retrieves the argument at key and converts it to type T.
// Returns (zero, false) if the key is missing, null, or incompatible.
//
// Example:
//
//	lat, ok := step.Args.Get[float64]("latitude")
//	role, ok := step.Args.Get[string]("role")
//	meta, ok := step.Args.Get[map[string]any]("metadata")
func (a Args) Get[T any](key string) (T, bool) {
	var zero T
	if a == nil {
		return zero, false
	}

	val, ok := a[key]
	if !ok || val == nil {
		return zero, false
	}

	if v, ok := val.(T); ok {
		return v, true
	}

	targetType := reflect.TypeOf(zero)
	if targetType == nil {
		return zero, false
	}

	switch targetType.Kind() {
	case reflect.String:
		switch s := val.(type) {
		case fmt.Stringer:
			return reflect.ValueOf(s.String()).Convert(targetType).Interface().(T), true
		default:
			str := fmt.Sprintf("%v", val)
			return reflect.ValueOf(str).Convert(targetType).Interface().(T), true
		}

	case reflect.Bool:
		switch b := val.(type) {
		case bool:
			return reflect.ValueOf(b).Convert(targetType).Interface().(T), true
		case string:
			if parsedBool, err := strconv.ParseBool(strings.TrimSpace(b)); err == nil {
				return reflect.ValueOf(parsedBool).Convert(targetType).Interface().(T), true
			}
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, ok := scalar.ToInt64(val); ok {
			return reflect.ValueOf(n).Convert(targetType).Interface().(T), true
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n, ok := scalar.ToInt64(val); ok && n >= 0 {
			return reflect.ValueOf(n).Convert(targetType).Interface().(T), true
		}

	case reflect.Float32, reflect.Float64:
		if f, ok := scalar.ToFloat64(val); ok {
			return reflect.ValueOf(f).Convert(targetType).Interface().(T), true
		}
	}

	return zero, false
}

// GetOr is like Get but returns a fallback if key is missing or null.
//
//	port := step.Args.GetOr("port", 8080)          // Inferred as int
//	host := step.Args.GetOr("host", "localhost")   // Inferred as string
//	priority := step.Args.GetOr("priority", false) // Inferred as bool
//	ratio := step.Args.GetOr("ratio", 0.95)        // Inferred as float64
func (a Args) GetOr[T any](key string, fallback T) T {
	if val, ok := a.Get[T](key); ok {
		return val
	}
	return fallback
}

// Slice retrieves an array of type T at key, safely coercing dynamic []any slices.
//
// Example:
//
//	tags := step.Args.Slice[string]("tags")
//	ids  := step.Args.Slice[int]("user_ids")
func (a Args) Slice[T any](key string) []T {
	if a == nil {
		return nil
	}

	val, ok := a[key]
	if !ok || val == nil {
		return nil
	}

	if raw, ok := val.([]T); ok {
		return raw
	}

	var zero T
	targetElemType := reflect.TypeOf(zero)
	if targetElemType == nil {
		return nil
	}

	coerceItem := func(item any) (T, bool) {
		tempArgs := Args{"_": item}
		return tempArgs.Get[T]("_")
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		res := make([]T, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			if elem == nil {
				continue
			}
			if coerced, ok := coerceItem(elem); ok {
				res = append(res, coerced)
			}
		}
		return res
	}

	return nil
}

// Bind marshals and unmarshals arguments directly into a destination struct.
func (a Args) Bind(dst any) error {
	if a == nil {
		return nil
	}
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("unmarshal args: %w", err)
	}
	return nil
}
