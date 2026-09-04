package core

import "math"

// Number represents all integer and floating-point numeric types.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// ToInt64 safely converts any primitive numeric type to an int64.
func ToInt64(val any) (int64, bool) {
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
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// ToFloat64 safely converts any primitive numeric type to a float64.
func ToFloat64(val any) (float64, bool) {
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
	default:
		return 0, false
	}
}

// CoerceNumber converts any numeric value into the requested target type T.
func CoerceNumber[T any](val any) (T, bool) {
	var zero T
	switch any(zero).(type) {
	case int:
		if n, ok := ToInt64(val); ok {
			return any(int(n)).(T), true
		}
	case int8:
		if n, ok := ToInt64(val); ok {
			return any(int8(n)).(T), true
		}
	case int16:
		if n, ok := ToInt64(val); ok {
			return any(int16(n)).(T), true
		}
	case int32:
		if n, ok := ToInt64(val); ok {
			return any(int32(n)).(T), true
		}
	case int64:
		if n, ok := ToInt64(val); ok {
			return any(n).(T), true
		}
	case uint:
		if n, ok := ToInt64(val); ok {
			return any(uint(n)).(T), true
		}
	case uint8:
		if n, ok := ToInt64(val); ok {
			return any(uint8(n)).(T), true
		}
	case uint16:
		if n, ok := ToInt64(val); ok {
			return any(uint16(n)).(T), true
		}
	case uint32:
		if n, ok := ToInt64(val); ok {
			return any(uint32(n)).(T), true
		}
	case uint64:
		if n, ok := ToInt64(val); ok {
			return any(uint64(n)).(T), true
		}
	case float32:
		if f, ok := ToFloat64(val); ok {
			return any(float32(f)).(T), true
		}
	case float64:
		if f, ok := ToFloat64(val); ok {
			return any(f).(T), true
		}
	}
	return zero, false
}
