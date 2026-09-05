package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
)

func writeManifestTree(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()

	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
			t.Fatalf("failed to create directory for %s: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write test file %s: %v", relPath, err)
		}
	}

	return tmpDir
}

func decodeSnippetSteps(t *testing.T, snippet string) ([]parser.ParsedStep, error) {
	t.Helper()

	file, diags := hclsyntax.ParseConfig([]byte(snippet), "snippet.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("syntax error in test snippet: %s", diags.Error())
	}

	pipeline := &parser.PipelineBlock{Body: file.Body}
	return parser.DecodePipelineSteps(pipeline)
}

func TestParse_DirectoryDiscovery(t *testing.T) {
	t.Parallel()

	t.Run("Single flat manifest file", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"main.hcl": `
endpoint "GET /health" {
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if len(manifest.Endpoints) != 1 || manifest.Endpoints[0].MethodAndPath != "GET /health" {
			t.Fatalf("expected 1 endpoint 'GET /health', got: %+v", manifest.Endpoints)
		}
	})

	t.Run("Nested multi-directory tree merged into unified AST", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"server.hcl": `
server {
  host = "0.0.0.0"
  port = 9000
}
`,
			"connections.hcl": `
connection "postgres" "primary" {
  source = "postgres://localhost/main"
}
`,
			"routes/v1/users.hcl": `
endpoint "GET /v1/users" {
  pipeline {
    respond {
      status = 200
    }
  }
}
endpoint "POST /v1/users" {
  pipeline {
    respond {
      status = 201
    }
  }
}
`,
			"routes/v2/orders.hcl": `
endpoint "GET /v2/orders" {
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if manifest.Server == nil || manifest.Server.Port != 9000 {
			t.Errorf("expected server port 9000, got: %+v", manifest.Server)
		}
		if len(manifest.Connections) != 1 || manifest.Connections[0].Driver != "postgres" {
			t.Errorf("expected 1 postgres connection, got: %+v", manifest.Connections)
		}
		if len(manifest.Endpoints) != 3 {
			t.Errorf("expected 3 merged endpoints, got %d", len(manifest.Endpoints))
		}
	})

	t.Run("Hidden directories and non-HCL files are ignored", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"routes.hcl": `
endpoint "GET /public" {
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
			".git/ignored.hcl": `
endpoint "GET /ignored" {
  pipeline {
    respond {
      status = 500
    }
  }
}
`,
			"README.md": "# Documentation",
			"init.sql":  "CREATE TABLE users();",
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if len(manifest.Endpoints) != 1 || manifest.Endpoints[0].MethodAndPath != "GET /public" {
			t.Fatalf("expected only 1 endpoint from routes.hcl, got: %+v", manifest.Endpoints)
		}
	})

	t.Run("Empty directory returns empty manifest without error", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manifest, err := parser.Parse(tmpDir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected error on empty directory: %v", err)
		}
		if len(manifest.Endpoints) != 0 || len(manifest.Connections) != 0 {
			t.Errorf("expected 0 endpoints and 0 connections, got: %+v", manifest)
		}
	})

	t.Run("Non-existent path returns descriptive error", func(t *testing.T) {
		t.Parallel()

		_, err := parser.Parse("path/does/not/exist", eval.BaseContext())
		if err == nil {
			t.Fatal("expected error on non-existent path, got nil")
		}
	})
}

func TestParse_SyntaxErrors(t *testing.T) {
	t.Parallel()

	t.Run("Reports file and line on syntax errors", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"bad.hcl": `
endpoint "GET /bad" {
  pipeline {
    respond = invalid syntax here !!!
  }
}
`,
		})

		_, err := parser.Parse(dir, eval.BaseContext())
		if err == nil {
			t.Fatal("expected syntax error, got nil")
		}
	})
}

func TestDecodePipelineSteps(t *testing.T) {
	t.Parallel()

	t.Run("Decodes go step with and without args", func(t *testing.T) {
		t.Parallel()

		snippet := `
go "auth" {
  use  = "auth.verify"
  args = { token = ctx.request.headers.authorization }
}
go "metrics" {
  use = "metrics.flush"
}
`
		steps, err := decodeSnippetSteps(t, snippet)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		if len(steps) != 2 {
			t.Fatalf("expected 2 steps, got %d", len(steps))
		}
		if steps[0].Type != parser.StepTypeGo || steps[0].Name != "auth" || steps[0].Go.Use != "auth.verify" {
			t.Errorf("step 0 mismatch: %+v", steps[0])
		}
		if steps[1].Type != parser.StepTypeGo || steps[1].Name != "metrics" || steps[1].Go.Use != "metrics.flush" {
			t.Errorf("step 1 mismatch: %+v", steps[1])
		}
	})

	t.Run("Decodes starlark step", func(t *testing.T) {
		t.Parallel()

		snippet := `
starlark "transform" {
  source = <<-STARLARK
    def execute(ctx):
      return {"count": len(ctx.request.body.get("items", []))}
  STARLARK
}
`
		steps, err := decodeSnippetSteps(t, snippet)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		if len(steps) != 1 || steps[0].Type != parser.StepTypeStarlark || steps[0].Name != "transform" {
			t.Errorf("step mismatch: %+v", steps)
		}
	})

	t.Run("Decodes sql step with query, args, and catch blocks", func(t *testing.T) {
		t.Parallel()

		snippet := `
sql "insert_user" {
  connection = connection.postgres.main
  query      = "INSERT INTO users (email) VALUES (@email) RETURNING id"
  args       = { email = ctx.request.body.email }

  catch "23505" {
    status  = 409
    headers = { "X-Error" = "Conflict" }
    body    = { error = "Email already exists" }
  }
}
`
		steps, err := decodeSnippetSteps(t, snippet)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}

		step := steps[0]
		if step.Type != parser.StepTypeSQL || step.Name != "insert_user" {
			t.Fatalf("step mismatch: %+v", step)
		}
		if step.SQL.Query != "INSERT INTO users (email) VALUES (@email) RETURNING id" {
			t.Errorf("unexpected query: %s", step.SQL.Query)
		}
		if len(step.SQL.Catches) != 1 || step.SQL.Catches[0].Code != "23505" {
			t.Errorf("catch block mismatch: %+v", step.SQL.Catches)
		}
	})

	t.Run("Decodes respond step with condition, status, headers, and body", func(t *testing.T) {
		t.Parallel()

		snippet := `
respond {
  condition = steps.find_user.rows_affected == 0
  status    = 404
  headers   = { "X-Trace" = "trace-123" }
  body      = { error = "Not found" }
}
`
		steps, err := decodeSnippetSteps(t, snippet)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		if len(steps) != 1 || steps[0].Type != parser.StepTypeRespond {
			t.Fatalf("expected respond step, got: %+v", steps)
		}
	})

	t.Run("Preserves declaration order across mixed step types", func(t *testing.T) {
		t.Parallel()

		snippet := `
sql "get_user" {
  connection = connection.postgres.main
  query      = "SELECT id FROM users"
}
starlark "sanitize" {
  source = "def execute(ctx): return {}"
}
go "notify" {
  use = "email.send"
}
respond {
  status = 200
  body   = steps.sanitize.result
}
`
		steps, err := decodeSnippetSteps(t, snippet)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}

		expectedOrder := []parser.StepType{
			parser.StepTypeSQL,
			parser.StepTypeStarlark,
			parser.StepTypeGo,
			parser.StepTypeRespond,
		}

		for i, expected := range expectedOrder {
			if steps[i].Type != expected {
				t.Errorf("step %d: expected %s, got %s", i, expected, steps[i].Type)
			}
		}
	})

	t.Run("Rejects unknown step types", func(t *testing.T) {
		t.Parallel()

		snippet := `
unsupported_block "foo" {
  attr = "bar"
}
`
		_, err := decodeSnippetSteps(t, snippet)
		if err == nil {
			t.Fatal("expected error on unknown step type, got nil")
		}
	})
}

func TestParse_BootTimeEvaluation(t *testing.T) {
	t.Setenv("APP_USER_PATTERN", "^[a-z0-9_]+$")

	dir := writeManifestTree(t, map[string]string{
		"schema.hcl": `
schema "account" {
  field "username" {
    type        = string
    pattern     = env("APP_USER_PATTERN")
    description = format("Minimum length is %d", 3)
  }
}
`,
	})

	manifest, err := parser.Parse(dir, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(manifest.Schemas) != 1 || len(manifest.Schemas[0].Fields) != 1 {
		t.Fatalf("expected 1 schema with 1 field, got: %+v", manifest.Schemas)
	}

	field := manifest.Schemas[0].Fields[0]
	if field.Pattern == nil || *field.Pattern != "^[a-z0-9_]+$" {
		t.Errorf("expected pattern '^[a-z0-9_]+$', got: %v", field.Pattern)
	}
	if field.Description == nil || *field.Description != "Minimum length is 3" {
		t.Errorf("expected description 'Minimum length is 3', got: %v", field.Description)
	}
}
