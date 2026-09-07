package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}

openapi {
  title       = "Store API"
  version     = "2.0.0"
  description = "Production API"
}

connection "postgres" "primary" {
  source = "postgres://user:pass@localhost:5432/db"
  pool {
    max_open     = 50
    max_lifetime = "1h"
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
  openapi "ui" {
    renderer = "scalar"
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

	docsEp := service.Endpoints[0]
	if docsEp.OpenAPI == nil || docsEp.OpenAPI.Renderer != "scalar" || docsEp.OpenAPI.Mode != compiler.OpenAPIModeUI {
		t.Errorf("expected scalar openapi handler, got: %+v", docsEp.OpenAPI)
	}
	if docsEp.OpenAPI.Title != "Store API" {
		t.Errorf("expected inherited title 'Store API', got %q", docsEp.OpenAPI.Title)
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
    respond { status = 200 }
  }
}
endpoint "GET /api/v1/users" {
  pipeline {
    respond { status = 200 }
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
  openapi "ui" { renderer = "scalar" }
  pipeline {
    respond { status = 200 }
  }
}
`,
			expectError: `endpoint "GET /api/v1/invalid" defines conflicting handlers: 'openapi "ui"' and 'pipeline'`,
		},
		{
			name: "Rejects endpoint with multiple openapi blocks",
			manifest: `
endpoint "GET /api/v1/multi-openapi" {
  openapi "spec" { format = "json" }
  openapi "ui" { renderer = "scalar" }
}
`,
			expectError: `endpoint "GET /api/v1/multi-openapi" defines multiple openapi handlers ("spec", "ui")`,
		},
		{
			name: "Rejects endpoint with request block and openapi handler",
			manifest: `
endpoint "GET /docs" {
  request {
    query {
      field "filter" { type = string }
    }
  }
  openapi "ui" { renderer = "scalar" }
}
`,
			expectError: `endpoint "GET /docs": openapi endpoints are engine-managed and do not accept a 'request' block`,
		},
		{
			name: "Rejects openapi endpoint with non-GET/HEAD method",
			manifest: `
endpoint "POST /openapi.json" {
  openapi "spec" { format = "json" }
}
`,
			expectError: `endpoint "POST /openapi.json" is invalid; openapi endpoints only support HTTP GET and HEAD`,
		},
		{
			name: "Rejects openapi with unknown mode",
			manifest: `
endpoint "GET /docs" {
  openapi "invalid_mode" {}
}
`,
			expectError: `endpoint "GET /docs": unsupported openapi mode "invalid_mode"`,
		},
		{
			name: "Rejects openapi ui with invalid renderer",
			manifest: `
endpoint "GET /docs" {
  openapi "ui" { renderer = "unknown_renderer" }
}
`,
			expectError: `endpoint "GET /docs": unsupported openapi renderer "unknown_renderer"`,
		},
		{
			name: "Rejects openapi spec with invalid format",
			manifest: `
endpoint "GET /openapi" {
  openapi "spec" { format = "xml" }
}
`,
			expectError: `endpoint "GET /openapi": invalid openapi format "xml"`,
		},
		{
			name: "Rejects openapi template with neither file nor inline",
			manifest: `
endpoint "GET /docs" {
  openapi "template" {}
}
`,
			expectError: `endpoint "GET /docs": openapi "template" requires exactly one of 'file' or 'inline'`,
		},
		{
			name: "Rejects openapi template with both file and inline",
			manifest: `
endpoint "GET /docs" {
  openapi "template" {
    file = "./test.html"
    inline = "<h1>hi</h1>"
  }
}
`,
			expectError: `endpoint "GET /docs": openapi "template" requires exactly one of 'file' or 'inline'`,
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
