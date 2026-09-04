package xstarlark_test

import (
	"reflect"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/ju4n97/hclapi/internal/steps/xstarlark"
)

func TestCaseInsensitiveDict_GoAPI(t *testing.T) {
	t.Parallel()

	t.Run("Empty dict properties and methods", func(t *testing.T) {
		t.Parallel()

		d := xstarlark.NewCaseInsensitiveDict(nil)

		if d.Len() != 0 {
			t.Errorf("d.Len() = %d; want 0", d.Len())
		}
		if d.Truth() != false {
			t.Errorf("d.Truth() = %v; want false", d.Truth())
		}
		if d.Type() != "case_insensitive_dict" {
			t.Errorf("d.Type() = %q; want 'case_insensitive_dict'", d.Type())
		}
		if d.String() != "{}" {
			t.Errorf("d.String() = %q; want '{}'", d.String())
		}
		if _, err := d.Hash(); err == nil {
			t.Error("d.Hash() succeeded; expected unhashable error")
		}

		// Non-string key lookup in Get
		val, found, err := d.Get(starlark.MakeInt(42))
		if err != nil || found || val != starlark.None {
			t.Errorf("Get(int) = (%v, %v, %v); want (None, false, nil)", val, found, err)
		}
	})

	t.Run("Populated dict from strings", func(t *testing.T) {
		t.Parallel()

		headers := map[string]string{
			"Authorization": "Bearer token123",
			"X-Api-Key":     "key456",
		}

		d := xstarlark.NewCaseInsensitiveDictFromStrings(headers)

		if d.Len() != 2 {
			t.Errorf("d.Len() = %d; want 2", d.Len())
		}
		if d.Truth() != true {
			t.Errorf("d.Truth() = %v; want true", d.Truth())
		}

		// Verify case-insensitive Go Get
		for _, key := range []string{"authorization", "Authorization", "AUTHORIZATION"} {
			val, found, err := d.Get(starlark.String(key))
			if err != nil || !found {
				t.Fatalf("Get(%q) found = %v, err = %v; want found = true", key, found, err)
			}
			if strVal, ok := val.(starlark.String); !ok || strVal.GoString() != "Bearer token123" {
				t.Errorf("Get(%q) = %v; want 'Bearer token123'", key, val)
			}
		}

		// ToMap verification
		exported := d.ToMap()
		if exported["authorization"] != "Bearer token123" || exported["x-api-key"] != "key456" {
			t.Errorf("ToMap() mismatch: %+v", exported)
		}
	})
}

func TestCaseInsensitiveDict_StarlarkHTTPHeaders(t *testing.T) {
	t.Parallel()

	headers := xstarlark.NewCaseInsensitiveDictFromStrings(map[string]string{
		"Authorization": "Bearer test-jwt",
		"X-Trace-Id":    "trace-abc-123",
	})

	ctx := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"request": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"headers": headers,
		}),
	})

	src := `
def execute(ctx):
    h = ctx.request.headers
    return {
        # 1. Bracket indexing (all variations)
        "bracket_lower": h["authorization"],
        "bracket_title": h["Authorization"],
        "bracket_upper": h["AUTHORIZATION"],
        "bracket_hyphen": h["X-Trace-Id"],

        # 2. .get() method (all variations)
        "get_lower": h.get("authorization"),
        "get_title": h.get("Authorization"),
        "get_upper": h.get("AUTHORIZATION"),
        "get_fallback": h.get("X-Missing-Header", "default_val"),

        # 3. 'in' membership operator
        "has_auth_title": "Authorization" in h,
        "has_auth_lower": "authorization" in h,
        "has_auth_upper": "AUTHORIZATION" in h,
        "has_missing": "X-Missing-Header" in h,

        # 4. Dot notation attribute access
        "dot_auth": h.authorization,

        # 5. len()
        "total_count": len(h),
    }
`

	res, err := xstarlark.Eval(src, ctx)
	if err != nil {
		t.Fatalf("Starlark execution failed: %v", err)
	}

	resultMap, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any result, got %T", res)
	}

	// Bracket indexing assertions
	for _, k := range []string{"bracket_lower", "bracket_title", "bracket_upper"} {
		if resultMap[k] != "Bearer test-jwt" {
			t.Errorf("%s = %v; want 'Bearer test-jwt'", k, resultMap[k])
		}
	}
	if resultMap["bracket_hyphen"] != "trace-abc-123" {
		t.Errorf("bracket_hyphen = %v; want 'trace-abc-123'", resultMap["bracket_hyphen"])
	}

	// .get() method assertions
	for _, k := range []string{"get_lower", "get_title", "get_upper"} {
		if resultMap[k] != "Bearer test-jwt" {
			t.Errorf("%s = %v; want 'Bearer test-jwt'", k, resultMap[k])
		}
	}
	if resultMap["get_fallback"] != "default_val" {
		t.Errorf("get_fallback = %v; want 'default_val'", resultMap["get_fallback"])
	}

	// 'in' operator assertions
	for _, k := range []string{"has_auth_title", "has_auth_lower", "has_auth_upper"} {
		if resultMap[k] != true {
			t.Errorf("%s = %v; want true", k, resultMap[k])
		}
	}
	if resultMap["has_missing"] != false {
		t.Errorf("has_missing = %v; want false", resultMap["has_missing"])
	}

	// Dot notation assertion
	if resultMap["dot_auth"] != "Bearer test-jwt" {
		t.Errorf("dot_auth = %v; want 'Bearer test-jwt'", resultMap["dot_auth"])
	}

	// len() assertion
	if resultMap["total_count"] != int64(2) {
		t.Errorf("total_count = %v; want 2", resultMap["total_count"])
	}
}

func TestCaseInsensitiveDict_StarlarkSQLRow(t *testing.T) {
	t.Parallel()

	// Simulates an SQL result row returned by Oracle/Postgres with mixed/uppercase identifiers
	sqlRow := xstarlark.NewCaseInsensitiveDict(map[string]any{
		"USER_ID":   int64(42),
		"USER_NAME": "Jane",
		"IS_ACTIVE": true,
		"BALANCE":   1250.75,
	})

	ctx := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"row": sqlRow,
	})

	src := `
def execute(ctx):
    r = ctx.row
    return {
        # Lowercase indexing against uppercase DB column
        "id_lower": r["user_id"],
        # Uppercase indexing
        "id_upper": r["USER_ID"],
        # Dot notation
        "id_dot": r.user_id,
        "name_dot": r.user_name,
        "active_dot": r.is_active,
        "balance_dot": r.balance,
        # Check dict methods
        "keys": sorted(r.keys()),
    }
`

	res, err := xstarlark.Eval(src, ctx)
	if err != nil {
		t.Fatalf("Starlark execution failed: %v", err)
	}

	resultMap, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any result, got %T", res)
	}

	if resultMap["id_lower"] != int64(42) {
		t.Errorf("id_lower = %v; want 42", resultMap["id_lower"])
	}
	if resultMap["id_upper"] != int64(42) {
		t.Errorf("id_upper = %v; want 42", resultMap["id_upper"])
	}
	if resultMap["id_dot"] != int64(42) {
		t.Errorf("id_dot = %v; want 42", resultMap["id_dot"])
	}
	if resultMap["name_dot"] != "Jane" {
		t.Errorf("name_dot = %v; want 'Jane'", resultMap["name_dot"])
	}
	if resultMap["active_dot"] != true {
		t.Errorf("active_dot = %v; want true", resultMap["active_dot"])
	}
	if resultMap["balance_dot"] != 1250.75 {
		t.Errorf("balance_dot = %v; want 1250.75", resultMap["balance_dot"])
	}

	// Verify keys were normalized to lowercase
	keys, ok := resultMap["keys"].([]any)
	if !ok {
		t.Fatalf("expected []any for keys, got %T", resultMap["keys"])
	}

	wantKeys := []any{"balance", "is_active", "user_id", "user_name"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Errorf("keys = %v; want %v", keys, wantKeys)
	}
}

func TestCaseInsensitiveDict_IterationAndMethods(t *testing.T) {
	t.Parallel()

	d := xstarlark.NewCaseInsensitiveDictFromStrings(map[string]string{
		"Primary":   "val1",
		"Secondary": "val2",
	})

	ctx := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"data": d,
	})

	src := `
def execute(ctx):
    d = ctx.data
    # Comprehension iteration
    iter_keys = [k for k in d]
    
    # .items() pairs
    items_list = [(k, v) for k, v in d.items()]
    
    return {
        "iter_keys": sorted(iter_keys),
        "values": sorted(d.values()),
        "items_len": len(items_list),
    }
`

	res, err := xstarlark.Eval(src, ctx)
	if err != nil {
		t.Fatalf("Starlark execution failed: %v", err)
	}

	resultMap := res.(map[string]any)

	gotKeys := resultMap["iter_keys"].([]any)
	wantKeys := []any{"primary", "secondary"}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Errorf("iter_keys = %v; want %v", gotKeys, wantKeys)
	}

	gotVals := resultMap["values"].([]any)
	wantVals := []any{"val1", "val2"}
	if !reflect.DeepEqual(gotVals, wantVals) {
		t.Errorf("values = %v; want %v", gotVals, wantVals)
	}

	if resultMap["items_len"] != int64(2) {
		t.Errorf("items_len = %v; want 2", resultMap["items_len"])
	}
}

func TestCaseInsensitiveDict_RoundTripToGo(t *testing.T) {
	t.Parallel()

	d := xstarlark.NewCaseInsensitiveDictFromStrings(map[string]string{
		"X-Custom": "payload",
	})

	ctx := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"headers": d,
	})

	// Script returns the CaseInsensitiveDict directly inside a return payload
	src := `
def execute(ctx):
    return {
        "echo_headers": ctx.headers,
    }
`

	res, err := xstarlark.Eval(src, ctx)
	if err != nil {
		t.Fatalf("Starlark execution failed: %v", err)
	}

	resultMap, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", res)
	}

	// Verify that StarlarkToGoValue unwrapped *CaseInsensitiveDict into a native Go map[string]any
	echoHeaders, ok := resultMap["echo_headers"].(map[string]any)
	if !ok {
		t.Fatalf(
			"expected echo_headers to be Go map[string]any, got %T (%v)",
			resultMap["echo_headers"],
			resultMap["echo_headers"],
		)
	}

	if echoHeaders["x-custom"] != "payload" {
		t.Errorf("echo_headers['x-custom'] = %v; want 'payload'", echoHeaders["x-custom"])
	}
}
