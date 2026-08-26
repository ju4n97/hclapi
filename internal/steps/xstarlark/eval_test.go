package xstarlark_test

import (
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/steps/xstarlark"
)

func TestEval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      string
		ctxData     map[string]any
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
			ctxData:     nil,
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
			name: "Context navigation with dot notation and list comprehension",
			source: `
def execute(ctx):
    prefix = ctx.request.headers.prefix
    tags = ctx.request.body.tags
    formatted = [prefix + ":" + t.strip().lower() for t in tags if len(t.strip()) > 0]
    return {
        "user": ctx.request.body.name.strip().upper(),
        "tags": formatted,
        "step_output": ctx.steps.prev_step.total
    }
`,
			ctxData: map[string]any{
				"request": map[string]any{
					"headers": map[string]string{"prefix": "tag"},
					"body": map[string]any{
						"name": "  jane  ",
						"tags": []string{" GO ", "", " PYTHON "},
					},
				},
				"steps": map[string]any{
					"prev_step": map[string]any{"total": 42},
				},
			},
			expectError: false,
			validate: func(t *testing.T, res any) {
				resMap := res.(map[string]any)
				if resMap["user"] != "JANE" {
					t.Errorf("expected 'JANE', got %v", resMap["user"])
				}
				if resMap["step_output"] != int64(42) {
					t.Errorf("expected 42, got %v", resMap["step_output"])
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
			ctxData:     nil,
			expectError: true,
			errorMsg:    "must define an 'execute(ctx)' function",
		},
		{
			name: "Execute is not a function returns error",
			source: `
execute = "this is a string, not a function"
`,
			ctxData:     nil,
			expectError: true,
			errorMsg:    "must be a callable function",
		},
		{
			name: "Syntax error in script returns error",
			source: `
def execute(ctx):
    invalid syntax here !
`,
			ctxData:     nil,
			expectError: true,
			errorMsg:    "compilation error",
		},
		{
			name: "Runtime error (division by zero) returns error",
			source: `
def execute(ctx):
    return 100 / 0
`,
			ctxData:     nil,
			expectError: true,
			errorMsg:    "runtime error",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := xstarlark.Eval(tt.source, tt.ctxData)

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
