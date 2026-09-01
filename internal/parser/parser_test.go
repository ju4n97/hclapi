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

// writeManifestTree writes an in-memory map of relative paths to file contents in a temp directory.
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

// decodeSnippetSteps parses a raw HCL pipeline block string and decodes its steps.
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
  url = "postgres://localhost/main"
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

func TestParse_Validation(t *testing.T) {
	t.Parallel()

	t.Run("Rejects duplicate connection keys across files", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"conn1.hcl": `connection "postgres" "primary" { url = "postgres://db1" }`,
			"conn2.hcl": `connection "postgres" "primary" { url = "postgres://db2" }`,
		})

		_, err := parser.Parse(dir, eval.BaseContext())
		if err == nil {
			t.Fatal("expected error on duplicate connection, got nil")
		}
		if !strings.Contains(err.Error(), `duplicate connection declaration "connection.postgres.primary"`) {
			t.Errorf("unexpected error message: %v", err)
		}
	})

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

func TestParse_SchemasAndRequests(t *testing.T) {
	t.Parallel()

	t.Run("Parses standalone schema with fields and constraints", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"schema.hcl": `
schema "user_create" {
  description = "User creation payload"

  field "email" {
    type        = string
    required    = true
    format      = "email"
    description = "Primary email address"
  }

  field "account_type" {
    type     = string
    required = true
    enum     = ["individual", "business"]
  }

  field "username" {
    type       = string
    required   = true
    min_length = 3
    max_length = 30
    pattern    = "^[a-zA-Z0-9_-]+$"
  }

  field "age" {
    type     = int
    required = false
    min      = 18
    max      = 120
  }

  field "tags" {
    type         = list(string)
    required     = false
    min_items    = 1
    max_items    = 10
    unique_items = true
  }
}
`,
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if len(manifest.Schemas) != 1 {
			t.Fatalf("expected 1 schema block, got %d", len(manifest.Schemas))
		}

		schema := manifest.Schemas[0]
		if schema.Name != "user_create" || *schema.Description != "User creation payload" {
			t.Errorf("unexpected schema name/description: %+v", schema)
		}
		if len(schema.Fields) != 5 {
			t.Fatalf("expected 5 fields, got %d", len(schema.Fields))
		}

		// Field 0: email
		if schema.Fields[0].Name != "email" || !schema.Fields[0].Required || *schema.Fields[0].Format != "email" {
			t.Errorf("field 0 mismatch: %+v", schema.Fields[0])
		}

		// Field 1: account_type with enum
		if schema.Fields[1].Name != "account_type" || schema.Fields[1].Enum == nil {
			t.Errorf("field 1 mismatch: %+v", schema.Fields[1])
		}

		// Field 2: username with bounds and pattern
		if schema.Fields[2].Name != "username" || *schema.Fields[2].MinLength != 3 || *schema.Fields[2].MaxLength != 30 ||
			*schema.Fields[2].Pattern != "^[a-zA-Z0-9_-]+$" {
			t.Errorf("field 2 mismatch: %+v", schema.Fields[2])
		}

		// Field 3: age with numeric bounds
		if schema.Fields[3].Name != "age" || *schema.Fields[3].Min != 18 || *schema.Fields[3].Max != 120 {
			t.Errorf("field 3 mismatch: %+v", schema.Fields[3])
		}

		// Field 4: tags with array bounds
		if schema.Fields[4].Name != "tags" || *schema.Fields[4].MinItems != 1 || *schema.Fields[4].MaxItems != 10 ||
			!schema.Fields[4].UniqueItems {
			t.Errorf("field 4 mismatch: %+v", schema.Fields[4])
		}
	})

	t.Run("Parses endpoint request with path, query, headers, and body reference", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"endpoint.hcl": `
endpoint "POST /users/{id}" {
  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }
    headers {
      field "x-api-key" {
        type     = string
        required = true
        format   = "uuid"
      }
    }
    query {
      field "referrer" {
        type     = string
        required = false
      }
    }
    body = schema.user_create
  }

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

		if len(manifest.Endpoints) != 1 {
			t.Fatalf("expected 1 endpoint, got %d", len(manifest.Endpoints))
		}

		req := manifest.Endpoints[0].Request
		if req == nil {
			t.Fatal("expected request block to be parsed")
		}

		if req.Path == nil || len(req.Path.Fields) != 1 || req.Path.Fields[0].Name != "id" {
			t.Errorf("path field group mismatch: %+v", req.Path)
		}
		if req.Headers == nil || len(req.Headers.Fields) != 1 || req.Headers.Fields[0].Name != "x-api-key" {
			t.Errorf("headers field group mismatch: %+v", req.Headers)
		}
		if req.Query == nil || len(req.Query.Fields) != 1 || req.Query.Fields[0].Name != "referrer" {
			t.Errorf("query field group mismatch: %+v", req.Query)
		}
		if req.BodyExpr == nil {
			t.Errorf("expected body reference expression to be parsed")
		}
	})

	t.Run("Parses endpoint request with inline body block", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"inline.hcl": `
endpoint "POST /webhooks" {
  request {
    body {
      field "event" {
        type     = string
        required = true
      }
      field "payload" {
        type     = any
        required = true
      }
    }
  }

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

		req := manifest.Endpoints[0].Request
		if req == nil || req.BodyInline == nil || len(req.BodyInline.Fields) != 2 {
			t.Fatalf("expected inline body with 2 fields, got: %+v", req)
		}
		if req.BodyInline.Fields[0].Name != "event" || req.BodyInline.Fields[1].Name != "payload" {
			t.Errorf("inline body fields mismatch: %+v", req.BodyInline.Fields)
		}
	})

	t.Run("Rejects duplicate schema declarations across files", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"schema1.hcl": `
schema "user" {
  field "name" {
    type = string
  }
}
`,
			"schema2.hcl": `
schema "user" {
  field "email" {
    type = string
  }
}
`,
		})

		_, err := parser.Parse(dir, eval.BaseContext())
		if err == nil {
			t.Fatal("expected error on duplicate schema name, got nil")
		}
		if !strings.Contains(err.Error(), `duplicate schema declaration "schema.user"`) {
			t.Errorf("unexpected error message: %v", err)
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
		t.Errorf("expected pattern from env() '^[a-z0-9_]+$', got: %v", field.Pattern)
	}
	if field.Description == nil || *field.Description != "Minimum length is 3" {
		t.Errorf("expected description from format() 'Minimum length is 3', got: %v", field.Description)
	}
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
		if steps[0].Go.Args == nil {
			t.Errorf("expected step 0 to have args expression")
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
		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}
		if steps[0].Type != parser.StepTypeStarlark || steps[0].Name != "transform" {
			t.Errorf("step mismatch: %+v", steps[0])
		}
		if !strings.Contains(steps[0].Starlark.Source, "def execute(ctx):") {
			t.Errorf("expected source to contain function definition: %s", steps[0].Starlark.Source)
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
			t.Fatalf("step type/name mismatch: %+v", step)
		}
		if step.SQL.Query != "INSERT INTO users (email) VALUES (@email) RETURNING id" {
			t.Errorf("unexpected query: %s", step.SQL.Query)
		}
		if step.SQL.Connection == nil {
			t.Errorf("expected connection and args expressions to be defined")
		}
		if step.SQL.Args == nil {
			t.Errorf("expected args expression to be defined")
		}
		if len(step.SQL.Catches) != 1 || step.SQL.Catches[0].Code != "23505" {
			t.Fatalf("expected 1 catch block with code '23505', got: %+v", step.SQL.Catches)
		}
		if step.SQL.Catches[0].Status == nil {
			t.Errorf("expected catch block status and body expressions to be defined")
		}
		if step.SQL.Catches[0].Headers == nil {
			t.Errorf("expected catch block status and body expressions to be defined")
		}
		if step.SQL.Catches[0].Body == nil {
			t.Errorf("expected catch block status and body expressions to be defined")
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
		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}

		step := steps[0]
		if step.Type != parser.StepTypeRespond {
			t.Fatalf("expected respond step, got: %s", step.Type)
		}
		if step.Respond.Condition == nil || step.Respond.Status == nil || step.Respond.Headers == nil ||
			step.Respond.Body == nil {
			t.Errorf("expected all respond attributes to be defined: %+v", step.Respond)
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
		if len(steps) != 4 {
			t.Fatalf("expected 4 steps, got %d", len(steps))
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

	t.Run("Rejects named steps missing required label", func(t *testing.T) {
		t.Parallel()

		snippet := `
go {
  use = "missing.label"
}
`
		_, err := decodeSnippetSteps(t, snippet)
		if err == nil {
			t.Fatal("expected error on go step without label, got nil")
		}
	})
}

func TestServerBlock_ToServer(t *testing.T) {
	t.Parallel()

	t.Run("Parses custom server settings and byte sizes", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"server.hcl": `
server {
  host          = "0.0.0.0"
  port          = 3000
  read_timeout  = "30s"
  max_body_size = "50MB"
}
`,
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		srv, err := manifest.Server.ToServer()
		if err != nil {
			t.Fatalf("unexpected mapping error: %v", err)
		}

		if srv.Host != "0.0.0.0" || srv.Port != 3000 {
			t.Errorf("unexpected host/port: %s:%d", srv.Host, srv.Port)
		}
		if srv.ReadTimeout.Duration() != 30*time.Second {
			t.Errorf("expected 30s read timeout, got %v", srv.ReadTimeout)
		}
		if srv.MaxBodySize.Bytes() != 50*1000*1000 {
			t.Errorf("expected 50MB, got %d", srv.MaxBodySize.Bytes())
		}
	})

	t.Run("Rejects malformed max_body_size unit", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"server.hcl": `
server {
  max_body_size = "10XB"
}
`,
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		_, err = manifest.Server.ToServer()
		if err == nil {
			t.Fatal("expected error on invalid byte size, got nil")
		}
	})
}

func TestConnectionBlock_ToConnection(t *testing.T) {
	t.Parallel()

	t.Run("Parses custom pool configuration and durations", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"connections.hcl": `
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
`,
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		conn, err := manifest.Connections[0].ToConnection()
		if err != nil {
			t.Fatalf("unexpected mapping error: %v", err)
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

	t.Run("Rejects malformed duration strings", func(t *testing.T) {
		t.Parallel()

		dir := writeManifestTree(t, map[string]string{
			"connections.hcl": `
connection "postgres" "bad_duration" {
  url = "postgres://localhost/db"
  pool {
    conn_max_lifetime = "invalid_duration"
  }
}
`,
		})

		manifest, err := parser.Parse(dir, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		_, err = manifest.Connections[0].ToConnection()
		if err == nil {
			t.Fatal("expected error on invalid duration, got nil")
		}
	})
}
