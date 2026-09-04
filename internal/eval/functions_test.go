package eval_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/runtime"
)

var (
	uuidV4Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uuidV7Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// System

func TestFunctions_System(t *testing.T) {
	t.Run("env", func(t *testing.T) {
		t.Setenv("HCLAPI_TEST_VAR", "active_value")

		// 1. Existing variable
		res, err := evalExpr(t, `env("HCLAPI_TEST_VAR")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "active_value" {
			t.Errorf("expected 'active_value', got %v", res)
		}

		// 2. Missing variable returns empty string
		resEmpty, err := evalExpr(t, `env("HCLAPI_NON_EXISTENT")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resEmpty != "" {
			t.Errorf("expected empty string, got %v", resEmpty)
		}

		// 3. Invalid argument type
		_, err = evalExpr(t, `env({ bad = true })`)
		if err == nil {
			t.Fatal("expected error when passing number to env(), got nil")
		}
	})

	t.Run("uuid and uuid_v4", func(t *testing.T) {
		t.Parallel()

		for _, fn := range []string{"uuid()", "uuid_v4()"} {
			res, err := evalExpr(t, fn)
			if err != nil {
				t.Fatalf("%s unexpected error: %v", fn, err)
			}
			strVal, ok := res.(string)
			if !ok {
				t.Fatalf("expected string, got %T", res)
			}
			if !uuidV4Regex.MatchString(strVal) {
				t.Errorf("%s returned %q which does not match UUID v4 format", fn, strVal)
			}
		}
	})

	t.Run("uuid_v7", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `uuid_v7()`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		strVal, ok := res.(string)
		if !ok {
			t.Fatalf("expected string, got %T", res)
		}
		if !uuidV7Regex.MatchString(strVal) {
			t.Errorf("uuid_v7() returned %q which does not match UUID v7 format", strVal)
		}
	})

	t.Run("now", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `now()`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		strVal, ok := res.(string)
		if !ok {
			t.Fatalf("expected string, got %T", res)
		}

		parsed, err := time.Parse(time.RFC3339, strVal)
		if err != nil {
			t.Fatalf("now() returned %q which failed RFC 3339 parsing: %v", strVal, err)
		}
		if time.Since(parsed) > 5*time.Second {
			t.Errorf("now() generated timestamp too far in the past: %v", parsed)
		}
	})

	t.Run("problem function builds RFC 9457 payload", func(t *testing.T) {
		t.Parallel()

		execCtx := &runtime.ExecutionContext{
			Server: manifest.Server{
				Problem: manifest.ProblemConfig{
					TypePrefix: "https://docs.example.com/errors/",
				},
			},
			RawRequest: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/42", http.NoBody),
		}

		// Standard status and detail
		res, err := eval.Any(parseExpr(t, `problem(404, "User not found")`), execCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", res)
		}

		if m["status"] != int64(404) || m["title"] != "Not Found" {
			t.Errorf("unexpected status/title: %+v", m)
		}
		if m["detail"] != "User not found" {
			t.Errorf("expected detail 'User not found', got %v", m["detail"])
		}
		if m["type"] != "https://docs.example.com/errors/not-found" {
			t.Errorf("expected custom base URL type, got %v", m["type"])
		}
		if m["instance"] != "/api/v1/users/42" {
			t.Errorf("expected instance '/api/v1/users/42', got %v", m["instance"])
		}

		// Default URN fallback without base URL
		execCtxNoBase := &runtime.ExecutionContext{
			RawRequest: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items", http.NoBody),
		}
		resURN, err := eval.Any(parseExpr(t, `problem(400, "Invalid parameter", "invalid-param")`), execCtxNoBase)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mURN := resURN.(map[string]any)
		if mURN["type"] != "urn:hclapi:error:invalid-param" {
			t.Errorf("expected URN type, got %v", mURN["type"])
		}
	})
}

// Encoding

func TestFunctions_Encoding(t *testing.T) {
	t.Parallel()

	t.Run("json_encode and json_decode round-trip", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `json_decode(json_encode({ id = 42, active = true, name = "jane" }))`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", res)
		}
		if m["id"] != int64(42) || m["active"] != true || m["name"] != "jane" {
			t.Errorf("unexpected decoded content: %+v", m)
		}
	})

	t.Run("json_decode malformed error", func(t *testing.T) {
		t.Parallel()

		_, err := evalExpr(t, `json_decode("{malformed json")`)
		if err == nil {
			t.Fatal("expected error on malformed JSON, got nil")
		}
	})

	t.Run("base64_encode and base64_decode", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `base64_decode(base64_encode("hello world!"))`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "hello world!" {
			t.Errorf("expected 'hello world!', got %v", res)
		}

		// Malformed base64
		_, err = evalExpr(t, `base64_decode("invalid!base64==")`)
		if err == nil {
			t.Fatal("expected error decoding invalid base64, got nil")
		}
	})

	t.Run("url_encode and url_decode", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `url_decode(url_encode("hello world & foo=bar"))`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "hello world & foo=bar" {
			t.Errorf("expected 'hello world & foo=bar', got %v", res)
		}
	})
}

// Strings

func TestFunctions_Strings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expr     string
		expected any
	}{
		{"lower", `lower("HELLO WORLD")`, "hello world"},
		{"upper", `upper("hello world")`, "HELLO WORLD"},
		{"trim_space", `trim_space("  hello \n\t")`, "hello"},
		{"trim", `trim("xxhelloxx", "x")`, "hello"},
		{"trim_prefix", `trim_prefix("Bearer token123", "Bearer ")`, "token123"},
		{"trim_suffix", `trim_suffix("filename.tar.gz", ".tar.gz")`, "filename"},
		{"replace", `replace("hello world", "world", "gopher")`, "hello gopher"},
		{"format", `format("Hello %s, score: %d", "Jane", 100)`, "Hello Jane, score: 100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := evalExpr(t, tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, res)
			}
		})
	}

	t.Run("split and join", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `join(":", split(",", "a,b,c"))`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "a:b:c" {
			t.Errorf("expected 'a:b:c', got %v", res)
		}
	})
}

// Collections & Objects

func TestFunctions_Collections(t *testing.T) {
	t.Parallel()

	t.Run("list", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			expr string
			want string
		}{
			{
				name: "string",
				expr: `list(string)`,
				want: "list(string)",
			},
			{
				name: "int",
				expr: `list(int)`,
				want: "list(int)",
			},
			{
				name: "float",
				expr: `list(float)`,
				want: "list(float)",
			},
			{
				name: "bool",
				expr: `list(bool)`,
				want: "list(bool)",
			},
			{
				name: "nested list",
				expr: `list(list(string))`,
				want: "list(list(string))",
			},
			{
				name: "nested map",
				expr: `list(map(string))`,
				want: "list(map(string))",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				res, err := evalExpr(t, tt.expr)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res != tt.want {
					t.Errorf("expected %q, got %q", tt.want, res)
				}
			})
		}
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			expr string
			want string
		}{
			{
				name: "string",
				expr: `map(string)`,
				want: "map(string)",
			},
			{
				name: "int",
				expr: `map(int)`,
				want: "map(int)",
			},
			{
				name: "float",
				expr: `map(float)`,
				want: "map(float)",
			},
			{
				name: "bool",
				expr: `map(bool)`,
				want: "map(bool)",
			},
			{
				name: "nested list",
				expr: `map(list(string))`,
				want: "map(list(string))",
			},
			{
				name: "nested map",
				expr: `map(map(string))`,
				want: "map(map(string))",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				res, err := evalExpr(t, tt.expr)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res != tt.want {
					t.Errorf("expected %q, got %q", tt.want, res)
				}
			})
		}
	})

	t.Run("coalesce", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `coalesce("", null, "first_valid", "fallback")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "first_valid" {
			t.Errorf("expected 'first_valid', got %v", res)
		}
	})

	t.Run("length", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			expr     string
			expected int64
		}{
			{`length(["a", "b", "c"])`, 3},
			{`length("gopher")`, 6},
			{`length({ a = 1, b = 2 })`, 2},
		}

		for _, tt := range tests {
			res, err := evalExpr(t, tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			if res != tt.expected {
				t.Errorf("for %s expected %d, got %v", tt.expr, tt.expected, res)
			}
		}
	})

	t.Run("merge", func(t *testing.T) {
		t.Parallel()

		res, err := evalExpr(t, `merge({ a = 1, b = 2 }, { b = 99, c = 3 })`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", res)
		}
		if m["a"] != int64(1) || m["b"] != int64(99) || m["c"] != int64(3) {
			t.Errorf("unexpected merge result: %+v", m)
		}
	})

	t.Run("lookup", func(t *testing.T) {
		t.Parallel()

		// 1. Key exists
		res1, err := evalExpr(t, `lookup({ role = "admin" }, "role", "user")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res1 != "admin" {
			t.Errorf("expected 'admin', got %v", res1)
		}

		// 2. Key missing -> returns default
		res2, err := evalExpr(t, `lookup({ role = "admin" }, "missing", "user")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res2 != "user" {
			t.Errorf("expected default 'user', got %v", res2)
		}
	})

	t.Run("keys and values", func(t *testing.T) {
		t.Parallel()

		resKeys, err := evalExpr(t, `keys({ name = "John", age = 30 })`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		kList := resKeys.([]any)
		if len(kList) != 2 || kList[0] != "age" || kList[1] != "name" {
			t.Errorf("unexpected keys result: %+v", kList)
		}
	})

	t.Run("contains", func(t *testing.T) {
		t.Parallel()

		resTrue, err := evalExpr(t, `contains(["admin", "member"], "admin")`)
		if err != nil || resTrue != true {
			t.Errorf("expected contains to be true, got %v (err: %v)", resTrue, err)
		}

		resFalse, err := evalExpr(t, `contains(["admin", "member"], "guest")`)
		if err != nil || resFalse != false {
			t.Errorf("expected contains to be false, got %v (err: %v)", resFalse, err)
		}
	})
}

// Cryptography

func TestFunctions_Cryptography(t *testing.T) {
	t.Parallel()

	t.Run("sha256", func(t *testing.T) {
		t.Parallel()

		// Known SHA-256 test vector for "hello"
		expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
		res, err := evalExpr(t, `sha256("hello")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != expected {
			t.Errorf("expected %s, got %v", expected, res)
		}
	})

	t.Run("md5", func(t *testing.T) {
		t.Parallel()

		// Known MD5 test vector for "hello"
		expected := "5d41402abc4b2a76b9719d911017c592"
		res, err := evalExpr(t, `md5("hello")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != expected {
			t.Errorf("expected %s, got %v", expected, res)
		}
	})

	t.Run("hmac_sha256", func(t *testing.T) {
		t.Parallel()

		secret := "my-secret-key"
		payload := "payload-data"

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(payload))
		expected := hex.EncodeToString(mac.Sum(nil))

		res, err := evalExpr(t, `hmac_sha256("my-secret-key", "payload-data")`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != expected {
			t.Errorf("expected %s, got %v", expected, res)
		}
	})
}

// Math

func TestFunctions_Math(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expr     string
		expected any
	}{
		{"min", `min(10, 5, 20, 3)`, int64(3)},
		{"max", `max(10, 5, 20, 3)`, int64(20)},
		{"abs negative", `abs(-42)`, int64(42)},
		{"abs positive", `abs(42)`, int64(42)},
		{"ceil", `ceil(4.2)`, int64(5)},
		{"floor", `floor(4.8)`, int64(4)},
		{"parse_int base 10", `parse_int("123", 10)`, int64(123)},
		{"parse_int base 16", `parse_int("ff", 16)`, int64(255)},
		{"parse_int base 2", `parse_int("1010", 2)`, int64(10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := evalExpr(t, tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, res)
			}
		})
	}

	t.Run("parse_int invalid string error", func(t *testing.T) {
		t.Parallel()

		_, err := evalExpr(t, `parse_int("invalid_number", 10)`)
		if err == nil {
			t.Fatal("expected error parsing invalid number, got nil")
		}
	})
}
