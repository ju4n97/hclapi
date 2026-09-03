package core

import (
	"reflect"
	"testing"
)

type customStringer struct {
	value string
}

func (c customStringer) String() string {
	return c.value
}

func TestArgs_Has(t *testing.T) {
	t.Parallel()

	args := Args{
		"name":     "jane",
		"null_val": nil,
		"count":    0,
		"enabled":  false,
	}

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "existing string", key: "name", want: true},
		{name: "existing zero integer", key: "count", want: true},
		{name: "existing false boolean", key: "enabled", want: true},
		{name: "existing nil value", key: "null_val", want: false},
		{name: "non-existent key", key: "missing", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := args.Has(tt.key); got != tt.want {
				t.Errorf("Args.Has(%q) = %v; want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestArgs_Get(t *testing.T) {
	t.Parallel()

	t.Run("direct type match", func(t *testing.T) {
		t.Parallel()
		args := Args{
			"str":  "hello",
			"bool": true,
			"map":  map[string]any{"nested": "value"},
		}

		if val, ok := args.Get[string]("str"); !ok || val != "hello" {
			t.Errorf("Get[string]() = (%v, %v); want (hello, true)", val, ok)
		}
		if val, ok := args.Get[bool]("bool"); !ok || val != true {
			t.Errorf("Get[bool]() = (%v, %v); want (true, true)", val, ok)
		}
		if val, ok := args.Get[map[string]any]("map"); !ok || val["nested"] != "value" {
			t.Errorf("Get[map[string]any]() = (%v, %v); want (map[nested:value], true)", val, ok)
		}
	})

	t.Run("HCL int64 numeric coercion", func(t *testing.T) {
		t.Parallel()
		// HCL AST evaluator produces int64 for all integer literals
		args := Args{"hcl_int": int64(42)}

		// Signed ints
		if val, ok := args.Get[int]("hcl_int"); !ok || val != 42 {
			t.Errorf("Get[int]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[int8]("hcl_int"); !ok || val != int8(42) {
			t.Errorf("Get[int8]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[int16]("hcl_int"); !ok || val != int16(42) {
			t.Errorf("Get[int16]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[int32]("hcl_int"); !ok || val != int32(42) {
			t.Errorf("Get[int32]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[int64]("hcl_int"); !ok || val != int64(42) {
			t.Errorf("Get[int64]() = (%v, %v); want (42, true)", val, ok)
		}

		// Unsigned ints
		if val, ok := args.Get[uint]("hcl_int"); !ok || val != uint(42) {
			t.Errorf("Get[uint]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[uint8]("hcl_int"); !ok || val != uint8(42) {
			t.Errorf("Get[uint8]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[uint16]("hcl_int"); !ok || val != uint16(42) {
			t.Errorf("Get[uint16]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[uint32]("hcl_int"); !ok || val != uint32(42) {
			t.Errorf("Get[uint32]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[uint64]("hcl_int"); !ok || val != uint64(42) {
			t.Errorf("Get[uint64]() = (%v, %v); want (42, true)", val, ok)
		}

		// Floats
		if val, ok := args.Get[float32]("hcl_int"); !ok || val != float32(42) {
			t.Errorf("Get[float32]() = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[float64]("hcl_int"); !ok || val != float64(42) {
			t.Errorf("Get[float64]() = (%v, %v); want (42, true)", val, ok)
		}
	})

	t.Run("HCL float64 numeric coercion", func(t *testing.T) {
		t.Parallel()
		args := Args{"hcl_float": float64(128.5)}

		if val, ok := args.Get[float64]("hcl_float"); !ok || val != 128.5 {
			t.Errorf("Get[float64]() = (%v, %v); want (128.5, true)", val, ok)
		}
		if val, ok := args.Get[float32]("hcl_float"); !ok || val != float32(128.5) {
			t.Errorf("Get[float32]() = (%v, %v); want (128.5, true)", val, ok)
		}
		// Truncation check
		if val, ok := args.Get[int]("hcl_float"); !ok || val != 128 {
			t.Errorf("Get[int]() truncated = (%v, %v); want (128, true)", val, ok)
		}
	})

	t.Run("string coercion", func(t *testing.T) {
		t.Parallel()
		args := Args{
			"num":      42,
			"boolean":  true,
			"stringer": customStringer{value: "custom-token"},
		}

		if val, ok := args.Get[string]("num"); !ok || val != "42" {
			t.Errorf("Get[string](int) = (%v, %v); want (42, true)", val, ok)
		}
		if val, ok := args.Get[string]("boolean"); !ok || val != "true" {
			t.Errorf("Get[string](bool) = (%v, %v); want (true, true)", val, ok)
		}
		if val, ok := args.Get[string]("stringer"); !ok || val != "custom-token" {
			t.Errorf("Get[string](fmt.Stringer) = (%v, %v); want (custom-token, true)", val, ok)
		}
	})

	t.Run("missing, nil, and incompatible values", func(t *testing.T) {
		t.Parallel()
		args := Args{
			"null_key":    nil,
			"invalid_num": "not-a-number",
		}

		// Ensure nil values don't coerce to the string literal "<nil>"
		if val, ok := args.Get[string]("null_key"); ok || val != "" {
			t.Errorf("Get[string](null_key) = (%q, %v); want (\"\", false)", val, ok)
		}
		if val, ok := args.Get[string]("missing"); ok || val != "" {
			t.Errorf("Get[string](missing) = (%q, %v); want (\"\", false)", val, ok)
		}
		if val, ok := args.Get[int]("null_key"); ok || val != 0 {
			t.Errorf("Get[int](null_key) = (%v, %v); want (0, false)", val, ok)
		}
		if val, ok := args.Get[int]("invalid_num"); ok || val != 0 {
			t.Errorf("Get[int](invalid_num) = (%v, %v); want (0, false)", val, ok)
		}
	})
}

func TestArgs_GetOr(t *testing.T) {
	t.Parallel()

	args := Args{
		"port":     int64(8080),
		"host":     "api.internal",
		"enabled":  true,
		"null_val": nil,
	}

	t.Run("inferred types using existing values", func(t *testing.T) {
		t.Parallel()
		// Test Go 1.27 type inference: no explicit [T] type parameter required
		if port := args.GetOr("port", 3000); port != 8080 {
			t.Errorf("GetOr(\"port\", 3000) = %v; want 8080", port)
		}
		if host := args.GetOr("host", "localhost"); host != "api.internal" {
			t.Errorf("GetOr(\"host\", ...) = %q; want \"api.internal\"", host)
		}
		if enabled := args.GetOr("enabled", false); enabled != true {
			t.Errorf("GetOr(\"enabled\", false) = %v; want true", enabled)
		}
	})

	t.Run("fallback used on missing or null keys", func(t *testing.T) {
		t.Parallel()
		if fallback := args.GetOr("missing_port", 9000); fallback != 9000 {
			t.Errorf("GetOr(missing) = %v; want 9000", fallback)
		}
		if fallback := args.GetOr("null_val", "default_val"); fallback != "default_val" {
			t.Errorf("GetOr(null_val) = %q; want \"default_val\"", fallback)
		}
		if fallback := args.GetOr("host", 1234); fallback != 1234 {
			// "api.internal" can't coerce to int, should return fallback
			t.Errorf("GetOr(incompatible) = %v; want 1234", fallback)
		}
	})
}

func TestArgs_Slice(t *testing.T) {
	t.Parallel()

	t.Run("exact typed slice", func(t *testing.T) {
		t.Parallel()
		args := Args{
			"strings": []string{"admin", "member"},
			"ints":    []int{10, 20, 30},
		}

		gotStr := args.Slice[string]("strings")
		wantStr := []string{"admin", "member"}
		if !reflect.DeepEqual(gotStr, wantStr) {
			t.Errorf("Slice[string] = %v; want %v", gotStr, wantStr)
		}

		gotInt := args.Slice[int]("ints")
		wantInt := []int{10, 20, 30}
		if !reflect.DeepEqual(gotInt, wantInt) {
			t.Errorf("Slice[int] = %v; want %v", gotInt, wantInt)
		}
	})

	t.Run("dynamic HCL []any slice with coercion", func(t *testing.T) {
		t.Parallel()
		// HCL AST evaluator produces []any with int64 numbers
		args := Args{
			"ids":   []any{int64(1), int64(2), int64(3)},
			"tags":  []any{"web", "prod", 404, nil},
			"mixed": []any{"10", 20.5, true},
		}

		// Coerce []any containing int64 to []int
		gotIDs := args.Slice[int]("ids")
		wantIDs := []int{1, 2, 3}
		if !reflect.DeepEqual(gotIDs, wantIDs) {
			t.Errorf("Slice[int] = %v; want %v", gotIDs, wantIDs)
		}

		// Coerce []any containing string/int/nil to []string (nil skipped)
		gotTags := args.Slice[string]("tags")
		wantTags := []string{"web", "prod", "404"}
		if !reflect.DeepEqual(gotTags, wantTags) {
			t.Errorf("Slice[string] = %v; want %v", gotTags, wantTags)
		}
	})

	t.Run("arbitrary slice via reflection fallback", func(t *testing.T) {
		t.Parallel()
		// Previous step passed a concrete []int64, but caller wants []int
		args := Args{
			"int64_slice": []int64{100, 200, 300},
		}

		got := args.Slice[int]("int64_slice")
		want := []int{100, 200, 300}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Slice[int](from []int64) = %v; want %v", got, want)
		}
	})

	t.Run("missing, nil, or non-slice keys", func(t *testing.T) {
		t.Parallel()
		args := Args{
			"not_a_slice": 42,
			"null_slice":  nil,
		}

		if got := args.Slice[string]("missing"); got != nil {
			t.Errorf("Slice(missing) = %v; want nil", got)
		}
		if got := args.Slice[string]("null_slice"); got != nil {
			t.Errorf("Slice(null) = %v; want nil", got)
		}
		if got := args.Slice[string]("not_a_slice"); got != nil {
			t.Errorf("Slice(non-slice) = %v; want nil", got)
		}
	})
}

func TestArgs_Bind(t *testing.T) {
	t.Parallel()

	type Config struct {
		Host    string   `json:"host"`
		Port    int      `json:"port"`
		Tags    []string `json:"tags"`
		Enabled bool     `json:"enabled"`
	}

	t.Run("successful struct binding", func(t *testing.T) {
		t.Parallel()
		args := Args{
			"host":    "0.0.0.0",
			"port":    8080,
			"tags":    []string{"api", "v1"},
			"enabled": true,
		}

		var cfg Config
		if err := args.Bind(&cfg); err != nil {
			t.Fatalf("Bind() failed unexpectedly: %v", err)
		}

		want := Config{
			Host:    "0.0.0.0",
			Port:    8080,
			Tags:    []string{"api", "v1"},
			Enabled: true,
		}

		if !reflect.DeepEqual(cfg, want) {
			t.Errorf("Bind() result = %+v; want %+v", cfg, want)
		}
	})

	t.Run("invalid destination pointer", func(t *testing.T) {
		t.Parallel()
		args := Args{"key": "value"}

		var nonPointer Config
		if err := args.Bind(nonPointer); err == nil {
			t.Error("Bind(nonPointer) succeeded; want error")
		}
	})
}

func TestArgs_NilReceiver(t *testing.T) {
	t.Parallel()

	// Ensure all methods are panic-safe when called on a nil Args map
	var a Args

	if a.Has("key") {
		t.Error("nil.Has() = true; want false")
	}

	if val, ok := a.Get[string]("key"); ok || val != "" {
		t.Errorf("nil.Get[string]() = (%q, %v); want (\"\", false)", val, ok)
	}

	if val := a.GetOr("port", 8080); val != 8080 {
		t.Errorf("nil.GetOr() = %v; want 8080", val)
	}

	if val := a.Slice[string]("tags"); val != nil {
		t.Errorf("nil.Slice() = %v; want nil", val)
	}

	var dummy struct{}
	if err := a.Bind(&dummy); err != nil {
		t.Errorf("nil.Bind() error = %v; want nil", err)
	}
}
