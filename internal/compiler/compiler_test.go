package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/compiler"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
)

func writeManifest(t *testing.T, content string) *parser.Manifest {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "manifest.hcl")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	manifest, err := parser.Parse(tmpDir, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	return manifest
}

func TestCompile_Success(t *testing.T) {
	t.Parallel()

	manifest := writeManifest(t, `
server {
  host          = "0.0.0.0"
  port          = 9000
  read_timeout  = "30s"
  max_body_size = "25MB"

  openapi {
    title       = "Store API"
    version     = "2.0.0"
    description = "Production API"
  }
}

connection "postgres" "primary" {
  url = "postgres://user:pass@localhost:5432/db"
  pool {
    max_open_conns    = 50
    conn_max_lifetime = "1h"
  }
}

schema "user_create" {
  field "email" {
    type     = string
    required = true
    format   = "email"
  }
}

endpoint "GET /docs" {
  openapi {
    ui = "scalar"
  }
}

endpoint "POST /api/v1/users" {
  request {
    headers {
      field "x-api-key" {
        type     = string
        required = true
        format   = "uuid"
      }
    }
    body = schema.user_create
  }

  pipeline {
    sql "insert_user" {
      connection = connection.postgres.primary
      query      = "INSERT INTO users (email) VALUES (@email)"
      args       = { email = ctx.request.body.email }
    }
    respond {
      status = 201
      body   = steps.insert_user.row
    }
  }
}
`)

	service, err := compiler.Compile(manifest, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected compilation error: %v", err)
	}

	// Verify Server compilation
	if service.Server.Host != "0.0.0.0" || service.Server.Port != 9000 {
		t.Errorf("unexpected server host/port: %+v", service.Server)
	}
	if service.Server.ReadTimeout.Duration() != 30*time.Second {
		t.Errorf("expected read_timeout 30s, got %v", service.Server.ReadTimeout)
	}
	if service.Server.MaxBodySize.Bytes() != 25*1000*1000 {
		t.Errorf("expected max_body_size 25MB, got %d", service.Server.MaxBodySize.Bytes())
	}
	if service.Server.OpenAPI.Title != "Store API" || service.Server.OpenAPI.Version != "2.0.0" {
		t.Errorf("unexpected openapi title/version: %+v", service.Server.OpenAPI)
	}

	// Verify Connections compilation
	if len(service.Connections) != 1 || service.Connections[0].Driver != "postgres" {
		t.Fatalf("expected 1 postgres connection, got: %+v", service.Connections)
	}
	if service.Connections[0].Pool.MaxOpenConns != 50 {
		t.Errorf("expected max_open_conns 50, got %d", service.Connections[0].Pool.MaxOpenConns)
	}

	// Verify Schemas compilation
	if len(service.Schemas) != 1 || len(service.Schemas["user_create"]) != 1 {
		t.Fatalf("expected 1 schema 'user_create', got: %+v", service.Schemas)
	}

	// Verify Endpoints compilation (1 docs endpoint + 1 API endpoint)
	if len(service.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(service.Endpoints))
	}

	// Docs endpoint
	docsEp := service.Endpoints[0]
	if docsEp.OpenAPI == nil || docsEp.OpenAPI.UI != "scalar" {
		t.Errorf("expected scalar openapi handler, got: %+v", docsEp.OpenAPI)
	}
	if docsEp.OpenAPI.Title != "Store API" {
		t.Errorf("expected inherited title 'Store API', got %q", docsEp.OpenAPI.Title)
	}

	// API endpoint
	apiEp := service.Endpoints[1]
	if len(apiEp.Rules.HeaderFields) != 1 || apiEp.Rules.HeaderFields[0].Name != "x-api-key" {
		t.Errorf("header rules mismatch: %+v", apiEp.Rules.HeaderFields)
	}
	if len(apiEp.Rules.BodyFields) != 1 || apiEp.Rules.BodyFields[0].Name != "email" {
		t.Errorf("body rules mismatch: %+v", apiEp.Rules.BodyFields)
	}
}

func TestCompile_ValidationFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		manifest    string
		expectError string
	}{
		{
			name: "Rejects duplicate endpoint routes",
			manifest: `
endpoint "GET /api/v1/users" {
  pipeline {
    respond {
      status = 200
    }
  }
}
endpoint "GET /api/v1/users" {
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
			expectError: `duplicate endpoint route "GET /api/v1/users"`,
		},
		{
			name: "Rejects empty pipeline without steps",
			manifest: `
endpoint "GET /api/v1/empty" {
  pipeline {}
}
`,
			expectError: `endpoint "GET /api/v1/empty": pipeline must declare at least one step`,
		},
		{
			name: "Rejects endpoint with both pipeline and openapi blocks",
			manifest: `
endpoint "GET /api/v1/invalid" {
  openapi { ui = "scalar" }
  pipeline {
    respond { status = 200 }
  }
}
`,
			expectError: `endpoint "GET /api/v1/invalid": cannot declare both pipeline and openapi blocks`,
		},
		{
			name: "Rejects endpoint without pipeline or openapi block",
			manifest: `
endpoint "GET /api/v1/missing-handler" {
  description = "Missing handler"
}
`,
			expectError: `endpoint "GET /api/v1/missing-handler": must declare either a pipeline or an openapi block`,
		},
		{
			name: "Rejects duplicate step names within a pipeline",
			manifest: `
endpoint "POST /api/v1/duplicate-steps" {
  pipeline {
    starlark "transform" {
      source = "def execute(ctx): return {}"
    }
    starlark "transform" {
      source = "def execute(ctx): return {}"
    }
    respond {
      status = 200
    }
  }
}
`,
			expectError: `endpoint "POST /api/v1/duplicate-steps": duplicate step name "transform" in pipeline`,
		},
		{
			name: "Rejects SQL step referencing unknown connection pool",
			manifest: `
endpoint "GET /api/v1/broken-conn" {
  pipeline {
    sql "fetch" {
      connection = connection.postgres.missing
      query      = "SELECT 1"
    }
    respond {
      status = 200
    }
  }
}
`,
			expectError: `endpoint "GET /api/v1/broken-conn": step "fetch": unknown connection "connection.postgres.missing"`,
		},
		{
			name: "Rejects request referencing non-existent body schema",
			manifest: `
endpoint "POST /api/v1/broken-schema" {
  request {
    body = schema.missing_schema
  }
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
			expectError: `endpoint "POST /api/v1/broken-schema": unknown schema reference "schema.missing_schema"`,
		},
		{
			name: "Rejects request referencing non-existent query schema",
			manifest: `
endpoint "GET /api/v1/broken-query" {
  request {
    query = schema.missing_pagination
  }
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
			expectError: `endpoint "GET /api/v1/broken-query": unknown schema reference "schema.missing_pagination"`,
		},
		{
			name: "Rejects duplicate schema declaration",
			manifest: `
schema "user" {
  field "name" {
    type = string
  }
}
schema "user" {
  field "email" {
    type = string
  }
}
`,
			expectError: `duplicate schema declaration "schema.user"`,
		},
		{
			name: "Rejects duplicate connection declaration",
			manifest: `
connection "postgres" "main" {
  url = "postgres://localhost/db1"
}
connection "postgres" "main" {
  url = "postgres://localhost/db2"
}
`,
			expectError: `duplicate connection declaration "connection.postgres.main"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manifest := writeManifest(t, tt.manifest)
			_, err := compiler.Compile(manifest, eval.BaseContext())
			if err == nil {
				t.Fatalf("expected compilation error containing %q, got nil", tt.expectError)
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected error to contain %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}
