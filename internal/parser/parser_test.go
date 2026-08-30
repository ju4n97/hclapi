package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/eval"
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
			targetPath:     "flat_single_file",
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
			targetPath:     "nested_tree",
			expectError:    false,
			expectedRoutes: 4, // 1 in root main.hcl + 2 in v1/users + 1 in v2/orders
		},
		{
			name:           "Case 04: Hidden directories (.git) are skipped completely",
			targetPath:     "hidden_dir_ignored",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:        "Case 05: Invalid HCL syntax / type mismatch",
			targetPath:  "syntax_error",
			expectError: true,
		},
		{
			name:        "Case 06: Missing required pipeline block",
			targetPath:  "missing_block",
			expectError: true,
		},
		{
			name:           "Case 07: Empty directory tree returns empty manifest without error",
			targetPath:     "empty_tree",
			expectError:    false,
			expectedRoutes: 0,
		},
		{
			name:           "Direct file target: Single HCL file path",
			targetPath:     "flat_single_file/main.hcl",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:           "Direct file target: Main file path",
			targetPath:     "nested_tree/main.hcl",
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
			manifest, err := parser.Parse(target, nil)

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

	parsePipeline := func(t *testing.T, hclSnippet string) parser.PipelineBlock {
		t.Helper()

		file, diags := hclsyntax.ParseConfig([]byte(hclSnippet), "snippet.hcl", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("test helper syntax error: %s", diags.Error())
		}
		return parser.PipelineBlock{Body: file.Body}
	}

	tests := []struct {
		name          string
		hclSnippet    string
		expectError   bool
		expectedSteps int
		validate      func(t *testing.T, steps []parser.ParsedStep)
	}{
		{
			name: "Valid sequential steps with correct order, labels, and dynamic expressions",
			hclSnippet: `
				go "step_one" {
					use  = "crypto.hash"
					args = { algorithm = "sha256" }
				}
				go "step_two" {
					use = "auth.verify"
				}
				respond {
					condition = steps.step_one.result != null
					status    = 200
					body      = "OK"
				}
			`,
			expectError:   false,
			expectedSteps: 3,
			validate: func(t *testing.T, steps []parser.ParsedStep) {
				// Step 0: Go with Args
				if steps[0].Type != parser.StepTypeGo || steps[0].Name != "step_one" ||
					steps[0].Go.Use != "crypto.hash" {
					t.Errorf("step 0 mismatch: %+v", steps[0])
				}
				val0, _ := steps[0].Go.Args.Value(nil)
				if val0.IsNull() {
					t.Errorf("expected step 0 args to be defined")
				}

				// Step 1: Go without Args
				if steps[1].Type != parser.StepTypeGo || steps[1].Name != "step_two" ||
					steps[1].Go.Use != "auth.verify" {
					t.Errorf("step 1 mismatch: %+v", steps[1])
				}
				if steps[1].Go.Args != nil {
					val1, _ := steps[1].Go.Args.Value(nil)
					if !val1.IsNull() {
						t.Errorf("expected step 1 args to be null/omitted, got: %v", val1)
					}
				}

				// Step 2: Respond
				if steps[2].Type != parser.StepTypeRespond {
					t.Errorf("step 2 mismatch: %+v", steps[2])
				}
				if steps[2].Respond.Condition == nil || steps[2].Respond.Status == nil || steps[2].Respond.Body == nil {
					t.Errorf("step 2 missing expressions: %+v", steps[2].Respond)
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
				if steps[1].Type != parser.StepTypeRespond {
					t.Errorf("step 1 mismatch: %+v", steps[1])
				}
				if steps[1].Respond.Body != nil {
					val, _ := steps[1].Respond.Body.Value(nil)
					if !val.IsNull() {
						t.Errorf("expected step 1 body to be null/omitted, got: %v", val)
					}
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

func TestConnectionBlockParsing(t *testing.T) {
	t.Parallel()

	t.Run("Parses driver, labels, evaluated url, and pool settings with durations", func(t *testing.T) {
		t.Parallel()

		snippet := `
connection "postgres" "primary" {
  url = "postgres://user:pass@localhost:5432/db"

  pool {
    max_open_conns    = 50
    max_idle_conns    = 10
    conn_max_lifetime = "1h"
    idle_timeout      = "10m"
    size              = 30
  }
}
`
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "conn.hcl"), []byte(snippet), 0o600); err != nil {
			t.Fatalf("failed to write test snippet: %v", err)
		}

		manifest, err := parser.Parse(tmpDir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if len(manifest.Connections) != 1 {
			t.Fatalf("expected 1 connection block, got %d", len(manifest.Connections))
		}

		conn, err := manifest.Connections[0].ToConnection()
		if err != nil {
			t.Fatalf("failed to map to connection: %v", err)
		}

		if conn.Driver != "postgres" || conn.Name != "primary" {
			t.Errorf("unexpected driver/name: %s.%s", conn.Driver, conn.Name)
		}
		if conn.Pool.MaxOpenConns != 50 || conn.Pool.MaxIdleConns != 10 {
			t.Errorf("unexpected open/idle conns: %+v", conn.Pool)
		}
		if conn.Pool.ConnMaxLifetime.Duration() != time.Hour {
			t.Errorf("expected 1h lifetime, got %v", conn.Pool.ConnMaxLifetime)
		}
		if conn.Pool.IdleTimeout.Duration() != 10*time.Minute {
			t.Errorf("expected 10m idle timeout, got %v", conn.Pool.IdleTimeout)
		}
	})

	t.Run("Rejects duplicate connection keys", func(t *testing.T) {
		t.Parallel()

		snippet := `
connection "postgres" "main" {
  url = "postgres://localhost/db1"
}
connection "postgres" "main" {
  url = "postgres://localhost/db2"
}
`
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "duplicate.hcl"), []byte(snippet), 0o600); err != nil {
			t.Fatalf("failed to write test snippet: %v", err)
		}

		_, err := parser.Parse(tmpDir, eval.BaseContext())
		if err == nil {
			t.Fatal("expected duplicate connection error, got nil")
		}
		if !strings.Contains(err.Error(), `duplicate connection declaration "connection.postgres.main"`) {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
