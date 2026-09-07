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
}

openapi {
  title       = "Store API"
  version     = "2.0.0"
  description = "Production API"
}

problem {
  type_prefix = "https://docs.example.com/errors/"
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

endpoint "GET /openapi.json" {
  openapi "spec" {
    format = "json"
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
	if service.OpenAPI.Title != "Store API" || service.OpenAPI.Version != "2.0.0" {
		t.Errorf("unexpected openapi title/version: %+v", service.OpenAPI)
	}
	if service.Problem.TypePrefix != "https://docs.example.com/errors/" {
		t.Errorf("unexpected problem prefix: %+v", service.Problem)
	}

	// Verify Connections compilation
	if len(service.Connections) != 1 || service.Connections[0].Driver != "postgres" {
		t.Fatalf("expected 1 postgres connection, got: %+v", service.Connections)
	}
	if service.Connections[0].Pool.MaxOpen != 50 {
		t.Errorf("expected max_open 50, got %d", service.Connections[0].Pool.MaxOpen)
	}

	// Verify Schemas compilation
	if len(service.Schemas) != 1 || len(service.Schemas["user_create"]) != 1 {
		t.Fatalf("expected 1 schema 'user_create', got: %+v", service.Schemas)
	}

	// Verify Endpoints compilation (1 spec endpoint + 1 docs endpoint + 1 API endpoint)
	if len(service.Endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(service.Endpoints))
	}

	var docsEp, specEp, apiEp *compiler.CompiledEndpoint
	for i := range service.Endpoints {
		switch service.Endpoints[i].MethodAndPath {
		case "GET /docs":
			docsEp = &service.Endpoints[i]
		case "GET /openapi.json":
			specEp = &service.Endpoints[i]
		case "POST /api/v1/users":
			apiEp = &service.Endpoints[i]
		}
	}

	if specEp == nil || specEp.OpenAPI == nil || specEp.OpenAPI.Mode != compiler.OpenAPIModeSpec {
		t.Fatalf("expected spec openapi handler, got: %+v", specEp)
	}

	// Docs endpoint auto-derivation check
	if docsEp == nil || docsEp.OpenAPI == nil || docsEp.OpenAPI.Renderer != "scalar" || docsEp.OpenAPI.Mode != compiler.OpenAPIModeUI {
		t.Fatalf("expected scalar openapi handler, got: %+v", docsEp)
	}
	if docsEp.OpenAPI.Title != "Store API" {
		t.Errorf("expected inherited title 'Store API', got %q", docsEp.OpenAPI.Title)
	}
	if docsEp.OpenAPI.SpecURL != "/openapi.json" {
		t.Errorf("expected auto-derived spec_url '/openapi.json', got %q", docsEp.OpenAPI.SpecURL)
	}

	// API endpoint
	if apiEp == nil {
		t.Fatalf("expected api endpoint to be present")
	}
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
  openapi "spec" {
    format = "json"
  }
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

func TestCompile_SpecURLAutoDerivation(t *testing.T) {
	t.Parallel()
	t.Run("Auto-derives single JSON spec endpoint", func(t *testing.T) {
		t.Parallel()
		manifest := writeManifest(t, `
openapi { title = "API" }
endpoint "GET /openapi.json" {
  openapi "spec" {
    format = "json"
  }
}
endpoint "GET /docs" {
  openapi "ui" {
    renderer = "scalar"
  }
}
`)
		svc, err := compiler.Compile(manifest, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected compile error: %v", err)
		}
		if svc.Endpoints[1].OpenAPI.SpecURL != "/openapi.json" {
			t.Errorf("expected spec_url '/openapi.json', got %q", svc.Endpoints[1].OpenAPI.SpecURL)
		}
	})
	t.Run("Disambiguates multiple JSON specs using path prefix", func(t *testing.T) {
		t.Parallel()
		manifest := writeManifest(t, `
openapi { title = "Versioned API" }
endpoint "GET /v1/openapi.json" {
  openapi "spec" {
    format = "json"
  }
}
endpoint "GET /v2/openapi.json" {
  openapi "spec" {
    format = "json"
  }
}
endpoint "GET /v1/docs" {
  openapi "ui" {
    renderer = "scalar"
  }
}
endpoint "GET /v2/docs" {
  openapi "ui" {
    renderer = "swagger"
  }
}
`)
		svc, err := compiler.Compile(manifest, eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected compile error: %v", err)
		}
		if svc.Endpoints[2].OpenAPI.SpecURL != "/v1/openapi.json" {
			t.Errorf("expected v1 spec_url '/v1/openapi.json', got %q", svc.Endpoints[2].OpenAPI.SpecURL)
		}
		if svc.Endpoints[3].OpenAPI.SpecURL != "/v2/openapi.json" {
			t.Errorf("expected v2 spec_url '/v2/openapi.json', got %q", svc.Endpoints[3].OpenAPI.SpecURL)
		}
	})
	t.Run("Fails when ambiguous and no common path prefix exists", func(t *testing.T) {
		t.Parallel()
		manifest := writeManifest(t, `
openapi { title = "Ambiguous API" }
endpoint "GET /openapi.json" {
  openapi "spec" {
    format = "json"
  }
}
endpoint "GET /api/spec.json" {
  openapi "spec" {
    format = "json"
  }
}
endpoint "GET /docs" {
  openapi "ui" {
    renderer = "scalar"
  }
}
`)
		_, err := compiler.Compile(manifest, eval.BaseContext())
		if err == nil || !strings.Contains(err.Error(), "ambiguous 'spec_url'") {
			t.Fatalf("expected ambiguous spec_url error, got: %v", err)
		}
	})
	t.Run("Fails when UI is declared with no JSON spec endpoint available", func(t *testing.T) {
		t.Parallel()
		manifest := writeManifest(t, `
openapi { title = "No Spec API" }
endpoint "GET /docs" {
  openapi "ui" {
    renderer = "scalar"
  }
}
`)
		_, err := compiler.Compile(manifest, eval.BaseContext())
		if err == nil || !strings.Contains(err.Error(), "no 'openapi \"spec\"' endpoint with format \"json\" found") {
			t.Fatalf("expected missing spec error, got: %v", err)
		}
	})
}
