package star_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/star"
)

func TestEval(t *testing.T) {
	t.Parallel()

	env := star.Env{
		Request: star.Request{
			Method: "POST",
			Path:   map[string]string{"id": "usr_100"},
			Query:  map[string]string{"limit": "10"},
			Headers: map[string]string{
				"Authorization": "Bearer secret_jwt",
				"X-Custom":      "value",
			},
			Body: map[string]any{
				"prefix": "item",
				"tags":   []any{" ALPHA ", "  ", "BETA"},
			},
		},
		Steps: map[string]map[string]any{
			"lookup": {
				"active": true,
				"count":  42,
			},
		},
		TimestampEpoch: 1771968000,
	}

	t.Run("Transforms request and prior step outputs", func(t *testing.T) {
		t.Parallel()

		script := `
def execute(ctx):
    prefix = ctx.request.body.get("prefix", "tag")
    raw_tags = ctx.request.body.get("tags", [])
    cleaned = [prefix + ":" + t.strip().lower() for t in raw_tags if len(t.strip()) > 0]

    return {
        "user_id": ctx.request.path["id"],
        "token": ctx.request.headers.get("authorization"),
        "dot_header": ctx.request.headers.authorization,
        "is_active": ctx.steps.lookup["active"],
        "tags": cleaned,
        "epoch": ctx.timestamp_epoch,
    }
`

		res, err := star.Eval(script, env)
		if err != nil {
			t.Fatalf("unexpected eval error: %v", err)
		}

		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", res)
		}

		if m["user_id"] != "usr_100" {
			t.Errorf("user_id = %v; want usr_100", m["user_id"])
		}
		if m["token"] != "Bearer secret_jwt" {
			t.Errorf("token = %v; want Bearer secret_jwt", m["token"])
		}
		if m["dot_header"] != "Bearer secret_jwt" {
			t.Errorf("dot_header = %v; want Bearer secret_jwt", m["dot_header"])
		}
		if m["is_active"] != true {
			t.Errorf("is_active = %v; want true", m["is_active"])
		}
		if m["epoch"] != int64(1771968000) {
			t.Errorf("epoch = %v; want 1771968000", m["epoch"])
		}

		tags := m["tags"].([]any)
		if len(tags) != 2 || tags[0] != "item:alpha" || tags[1] != "item:beta" {
			t.Errorf("unexpected tags: %+v", tags)
		}
	})

	t.Run("Fails with CompileError on invalid syntax", func(t *testing.T) {
		t.Parallel()

		_, err := star.Eval("def execute(ctx): invalid syntax here", env)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var compErr *star.CompileError
		if !errors.As(err, &compErr) {
			t.Errorf("expected *star.CompileError, got %T (%v)", err, err)
		}
	})

	t.Run("Fails when execute(ctx) is not defined", func(t *testing.T) {
		t.Parallel()

		_, err := star.Eval("def helper(): return 42", env)
		if !errors.Is(err, star.ErrMissingExecute) {
			t.Errorf("expected ErrMissingExecute, got %v", err)
		}
	})

	t.Run("Aborts infinite while loops via step limit", func(t *testing.T) {
		t.Parallel()

		script := `
def execute(ctx):
    x = 0
    while True:
        x += 1
    return x
`
		_, err := star.EvalWithLimit(script, env, 10_000)
		if err == nil {
			t.Fatal("expected step limit error, got nil")
		}

		var runErr *star.RuntimeError
		if !errors.As(err, &runErr) {
			t.Fatalf("expected *star.RuntimeError, got %T", err)
		}
		if !strings.Contains(runErr.Error(), "too many steps") {
			t.Errorf("expected 'too many steps' message, got %q", runErr.Error())
		}
	})
}
