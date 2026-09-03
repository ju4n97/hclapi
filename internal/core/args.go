package core

import (
	"encoding/json"
	"fmt"
	"reflect"
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

	// Exact type match
	if v, ok := val.(T); ok {
		return v, true
	}

	// Numeric coercion
	if num, ok := coerceNumber[T](val); ok {
		return num, true
	}

	// String coercion if T is string
	if _, isString := any(zero).(string); isString {
		switch s := val.(type) {
		case fmt.Stringer:
			return any(s.String()).(T), true
		default:
			return any(fmt.Sprintf("%v", val)).(T), true
		}
	}

	return zero, false
}

// GetOr is like Get but returns a fallback if key is missing or null.
//
//	port := step.Args.GetOr("port", 8080)            // Inferred as int
//	host := step.Args.GetOr("host", "localhost")     // Inferred as string
//	priority := step.Args.GetOr("priority", false)   // Inferred as bool
//	ratio := step.Args.GetOr("ratio", 0.95)          // Inferred as float64
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

	// Exact slice type match
	if raw, ok := val.([]T); ok {
		return raw
	}

	var zero T
	_, isString := any(zero).(string)

	// Default dynamic slice ([]any)
	if raw, ok := val.([]any); ok {
		res := make([]T, 0, len(raw))
		for _, item := range raw {
			if item == nil {
				continue
			}
			if v, ok := item.(T); ok {
				res = append(res, v)
			} else if num, ok := coerceNumber[T](item); ok {
				res = append(res, num)
			} else if isString {
				res = append(res, any(fmt.Sprintf("%v", item)).(T))
			}
		}
		return res
	}

	// Fallback: arbitrary slice types, like []int64 from prior Go step
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Slice {
		res := make([]T, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			item := rv.Index(i).Interface()
			if item == nil {
				continue
			}
			if v, ok := item.(T); ok {
				res = append(res, v)
			} else if num, ok := coerceNumber[T](item); ok {
				res = append(res, num)
			} else if isString {
				res = append(res, any(fmt.Sprintf("%v", item)).(T))
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

func coerceNumber[T any](val any) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case int:
		if n, ok := toInt64(val); ok {
			return any(int(n)).(T), true
		}
	case int8:
		if n, ok := toInt64(val); ok {
			return any(int8(n)).(T), true
		}
	case int16:
		if n, ok := toInt64(val); ok {
			return any(int16(n)).(T), true
		}
	case int32:
		if n, ok := toInt64(val); ok {
			return any(int32(n)).(T), true
		}
	case int64:
		if n, ok := toInt64(val); ok {
			return any(n).(T), true
		}
	case uint:
		if n, ok := toInt64(val); ok {
			return any(uint(n)).(T), true
		}
	case uint8:
		if n, ok := toInt64(val); ok {
			return any(uint8(n)).(T), true
		}
	case uint16:
		if n, ok := toInt64(val); ok {
			return any(uint16(n)).(T), true
		}
	case uint32:
		if n, ok := toInt64(val); ok {
			return any(uint32(n)).(T), true
		}
	case uint64:
		if n, ok := toInt64(val); ok {
			return any(uint64(n)).(T), true
		}
	case float32:
		if f, ok := toFloat64(val); ok {
			return any(float32(f)).(T), true
		}
	case float64:
		if f, ok := toFloat64(val); ok {
			return any(f).(T), true
		}
	}
	return zero, false
}

func toInt64(val any) (int64, bool) {
	switch v := val.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	}
	return 0, false
}

func toFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	}
	return 0, false
}
