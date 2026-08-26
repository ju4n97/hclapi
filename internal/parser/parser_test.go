package parser_test

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/ju4n97/hclapi/internal/parser"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		targetPath     string
		expectError    bool
		expectedRoutes int
		validate       func(t *testing.T, m *parser.Manifest)
	}{
		{
			name:           "Case 01: Flat single file in directory",
			targetPath:     "01_flat_single_file",
			expectError:    false,
			expectedRoutes: 1,
			validate: func(t *testing.T, m *parser.Manifest) {
				ep := m.Endpoints[0]
				if ep.MethodAndPath != "GET /health" {
					t.Errorf("expected MethodAndPath 'GET /health', got %q", ep.MethodAndPath)
				}
			},
		},
		{
			name:           "Case 02: Deeply nested directory tree merged into one AST",
			targetPath:     "02_nested_tree",
			expectError:    false,
			expectedRoutes: 4, // 1 in root Hclapifile + 2 in v1/users + 1 in v2/orders
		},
		{
			name:           "Case 03: Mixed extensions (Hclapifile, .hclapi, .hcl) and ignores non-manifests",
			targetPath:     "03_mixed_extensions",
			expectError:    false,
			expectedRoutes: 3,
		},
		{
			name:           "Case 04: Hidden directories (.git) are skipped completely",
			targetPath:     "04_hidden_dir_ignored",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:        "Case 05: Invalid HCL syntax / type mismatch",
			targetPath:  "05_syntax_error",
			expectError: true,
		},
		{
			name:        "Case 06: Missing required pipeline block",
			targetPath:  "06_missing_block",
			expectError: true,
		},
		{
			name:           "Case 07: Empty directory tree returns empty manifest without error",
			targetPath:     "07_empty_tree",
			expectError:    false,
			expectedRoutes: 0,
		},
		{
			name:           "Direct file target: Single HCL file path",
			targetPath:     "01_flat_single_file/main.hcl",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:           "Direct file target: Extensionless Hclapifile path",
			targetPath:     "02_nested_tree/Hclapifile",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:        "Non-existent path returns error",
			targetPath:  "does_not_exist",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target := filepath.Join("testdata", tt.targetPath)
			manifest, err := parser.Parse(target)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected an error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if manifest == nil {
				t.Fatalf("expected manifest to not be nil")
			}

			if len(manifest.Endpoints) != tt.expectedRoutes {
				t.Errorf("expected %d routes, got %d", tt.expectedRoutes, len(manifest.Endpoints))
			}

			if tt.validate != nil {
				tt.validate(t, manifest)
			}
		})
	}
}

func TestDecodePipelineSteps(t *testing.T) {
	t.Parallel()

	// Helper to create a parser.Pipeline from an HCL snippet
	parsePipeline := func(t *testing.T, hclSnippet string) parser.Pipeline {
		t.Helper()

		file, diags := hclsyntax.ParseConfig([]byte(hclSnippet), "snippet.hcl", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("test helper syntax error: %s", diags.Error())
		}
		return parser.Pipeline{Body: file.Body}
	}

	tests := []struct {
		name          string
		hclSnippet    string
		expectError   bool
		expectedSteps int
		validate      func(t *testing.T, steps []parser.ParsedStep)
	}{
		{
			name: "Valid sequential steps with correct order and labels",
			hclSnippet: `
				go "step_one" {
					use = "crypto.hash"
				}
				go "step_two" {
					use = "auth.verify"
				}
				respond {
					status = 200
					body   = "OK"
				}
			`,
			expectError:   false,
			expectedSteps: 3,
			validate: func(t *testing.T, steps []parser.ParsedStep) {
				if steps[0].Type != parser.StepTypeGo || steps[0].Name != "step_one" || steps[0].Go.Use != "crypto.hash" {
					t.Errorf("step 0 mismatch: %+v", steps[0])
				}
				if steps[1].Type != parser.StepTypeGo || steps[1].Name != "step_two" || steps[1].Go.Use != "auth.verify" {
					t.Errorf("step 1 mismatch: %+v", steps[1])
				}
				if steps[2].Type != parser.StepTypeRespond || steps[2].Respond.Status != 200 || *steps[2].Respond.Body != "OK" {
					t.Errorf("step 2 mismatch: %+v", steps[2])
				}
			},
		},
		{
			name:          "Empty pipeline body returns 0 steps without error",
			hclSnippet:    ``,
			expectError:   false,
			expectedSteps: 0,
		},
		{
			name: "Unknown step type returns error",
			hclSnippet: `
				unsupported_step "foo" {
					some_attr = "bar"
				}
			`,
			expectError: true,
		},
		{
			name: "Invalid attribute type in go step returns error",
			hclSnippet: `
				go "bad_step" {
					use = ["cannot", "be", "a", "list"] # HCL can't coerce a list to string
				}
			`,
			expectError: true,
		},
		{
			name: "Invalid attribute type in respond step returns error",
			hclSnippet: `
				respond {
					status = "not-an-int"
				}
			`,
			expectError: true,
		},
		{
			name: "Go step without required label returns error",
			hclSnippet: `
				go {
					use = "logger.flush"
				}
			`,
			expectError: true,
		},
		{
			name: "Valid starlark step decoding",
			hclSnippet: `
				starlark "transform" {
					source = "def execute(ctx): return {'ok': True}"
				}
				respond {
					status = 200
				}
			`,
			expectError:   false,
			expectedSteps: 2,
			validate: func(t *testing.T, steps []parser.ParsedStep) {
				if steps[0].Type != parser.StepTypeStarlark || steps[0].Name != "transform" {
					t.Errorf("step 0 mismatch: %+v", steps[0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := parsePipeline(t, tt.hclSnippet)
			steps, err := parser.DecodePipelineSteps(&p)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(steps) != tt.expectedSteps {
				t.Errorf("expected %d steps, got %d", tt.expectedSteps, len(steps))
			}

			if tt.validate != nil {
				tt.validate(t, steps)
			}
		})
	}
}
