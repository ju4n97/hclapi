package star_test

import (
	"reflect"
	"testing"

	"go.starlark.net/starlark"

	"github.com/ju4n97/hclapi/internal/star"
)

func TestCaseInsensitiveDict(t *testing.T) {
	t.Parallel()

	headers := map[string]string{
		"Authorization": "Bearer token123",
		"X-Trace-Id":    "trace-abc",
	}

	d := star.NewCaseInsensitiveDictFromStrings(headers)

	t.Run("Bracket indexing is case-insensitive", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"authorization", "Authorization", "AUTHORIZATION"} {
			val, found, err := d.Get(starlark.String(key))
			if err != nil || !found {
				t.Fatalf("expected key %q to be found", key)
			}
			if val.(starlark.String).GoString() != "Bearer token123" {
				t.Errorf("expected 'Bearer token123', got %v", val)
			}
		}
	})

	t.Run("Missing key returns false and None", func(t *testing.T) {
		t.Parallel()

		val, found, err := d.Get(starlark.String("X-Missing"))
		if err != nil || found || val != starlark.None {
			t.Errorf("expected (None, false, nil), got (%v, %v, %v)", val, found, err)
		}
	})

	t.Run("ToMap returns lowercase keys", func(t *testing.T) {
		t.Parallel()

		m := d.ToMap()
		expected := map[string]any{
			"authorization": "Bearer token123",
			"x-trace-id":    "trace-abc",
		}
		if !reflect.DeepEqual(m, expected) {
			t.Errorf("got %+v; want %+v", m, expected)
		}
	})
}
