package xstarlark_test

import (
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/ju4n97/hclapi/internal/steps/xstarlark"
)

func TestEval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      string
		ctx         starlark.Value
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, result any)
	}{
		{
			name: "Basic arithmetic and dictionary return",
			source: `
def execute(ctx):
    return {"calculated": 10 + 5, "active": True}
`,
			ctx:         starlark.None,
			expectError: false,
			validate: func(t *testing.T, res any) {
				resMap, ok := res.(map[string]any)
				if !ok {
					t.Fatalf("expected map[string]any, got %T", res)
				}
				if resMap["calculated"] != int64(15) {
					t.Errorf("expected 15, got %v", resMap["calculated"])
				}
				if resMap["active"] != true {
					t.Errorf("expected true, got %v", resMap["active"])
				}
			},
		},
		{
			name: "Idiomatic Starlark: envelope struct with dictionary subscript and .get()",
			source: `
def execute(ctx):
    prefix = ctx.request.headers.get("prefix", "tag")
    tags = ctx.request.body.get("tags", [])
    user_name = ctx.request.body["name"].strip().upper()
    formatted = [prefix + ":" + t.strip().lower() for t in tags if len(t.strip()) > 0]
    return {
        "user": user_name,
        "tags": formatted,
        "step_output": ctx.steps.prev_step["total"],
        "timestamp": ctx.timestamp_epoch
    }
`,
			ctx: starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"timestamp_epoch": starlark.MakeInt64(1700000000),
				"request": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
					"headers": xstarlark.GoToStarlarkValue(map[string]string{"prefix": "tag"}),
					"body": xstarlark.GoToStarlarkValue(map[string]any{
						"name": "  jane  ",
						"tags": []string{" GO ", "", " PYTHON "},
					}),
				}),
				"steps": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
					"prev_step": xstarlark.GoToStarlarkValue(map[string]any{"total": 42}),
				}),
			}),
			expectError: false,
			validate: func(t *testing.T, res any) {
				resMap := res.(map[string]any)
				if resMap["user"] != "JANE" {
					t.Errorf("expected 'JANE', got %v", resMap["user"])
				}
				if resMap["step_output"] != int64(42) {
					t.Errorf("expected 42, got %v", resMap["step_output"])
				}
				if resMap["timestamp"] != int64(1700000000) {
					t.Errorf("expected 1700000000, got %v", resMap["timestamp"])
				}
				tags := resMap["tags"].([]any)
				if len(tags) != 2 || tags[0] != "tag:go" || tags[1] != "tag:python" {
					t.Errorf("unexpected tags: %v", tags)
				}
			},
		},
		{
			name: "Missing execute function returns error",
			source: `
def helper():
    return 42
`,
			ctx:         starlark.None,
			expectError: true,
			errorMsg:    "must define an 'execute(ctx)' function",
		},
		{
			name: "Syntax error in script returns error",
			source: `
def execute(ctx):
    invalid syntax here !
`,
			ctx:         starlark.None,
			expectError: true,
			errorMsg:    "compilation error",
		},
		{
			name: "Runtime error (division by zero) returns error",
			source: `
def execute(ctx):
    return 100 / 0
`,
			ctx:         starlark.None,
			expectError: true,
			errorMsg:    "runtime error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := xstarlark.Eval(tt.source, tt.ctx)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorMsg)
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, res)
			}
		})
	}
}
