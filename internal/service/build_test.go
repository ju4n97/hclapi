package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/service"
)

func writeManifest(t *testing.T, content string) *manifest.Manifest {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.hcl")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	m, err := manifest.Load(dir, eval.BaseContext())
	if err != nil {
		t.Fatalf("failed to load test manifest: %v", err)
	}
	return m
}

func TestBuild_Success(t *testing.T) {
	t.Parallel()

	hcl := `
server {
  host          = "0.0.0.0"
  port          = 9000
  read_timeout  = "30s"
  write_timeout = "20s"
  idle_timeout  = "90s"
  max_body_size = "25MB"
}

openapi {
  title       = "Storefront API"
  version     = "2.0.0"
  description = "Production service API"
}

problem {
  type_prefix = "https://docs.example.com/errors/"
}

connection "postgres" "primary" {
  source = "postgres://user:pass@localhost:5432/db"
  pool {
    max_open     = 50
    max_idle     = 10
    max_lifetime = "1h"
    idle_timeout = "15m"
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
`

	m := writeManifest(t, hcl)
	svc, err := service.Build(m, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if svc.Server.Host != "0.0.0.0" || svc.Server.Port != 9000 {
		t.Errorf("unexpected server host/port: %+v", svc.Server)
	}
	if svc.Server.ReadTimeout != 30*time.Second {
		t.Errorf("expected 30s read_timeout, got %v", svc.Server.ReadTimeout)
	}
	if svc.Server.WriteTimeout != 20*time.Second {
		t.Errorf("expected 20s write_timeout, got %v", svc.Server.WriteTimeout)
	}
	if svc.Server.IdleTimeout != 90*time.Second {
		t.Errorf("expected 90s idle_timeout, got %v", svc.Server.IdleTimeout)
	}
	if svc.Server.MaxBodySize != 25*1000*1000 {
		t.Errorf("expected 25MB max_body_size, got %d", svc.Server.MaxBodySize)
	}

	if svc.Problem.TypePrefix != "https://docs.example.com/errors/" {
		t.Errorf("unexpected problem prefix: %q", svc.Problem.TypePrefix)
	}
	if got := svc.Problem.FormatType("not-found"); got != "https://docs.example.com/errors/not-found" {
		t.Errorf("unexpected FormatType: %q", got)
	}

	if len(svc.Connections) != 1 || svc.Connections[0].Key() != "postgres.primary" {
		t.Fatalf("expected postgres.primary connection, got: %+v", svc.Connections)
	}
	if svc.Connections[0].Pool.MaxOpen != 50 || svc.Connections[0].Pool.MaxLifetime != time.Hour {
		t.Errorf("unexpected connection pool settings: %+v", svc.Connections[0].Pool)
	}

	userSchema, ok := svc.Schemas["user_create"]
	if !ok || len(userSchema.Fields) != 1 {
		t.Fatalf("expected schema user_create with 1 field, got: %+v", svc.Schemas)
	}
	if userSchema.Fields[0].Name != "email" || userSchema.Fields[0].Type != "string" || !userSchema.Fields[0].Required {
		t.Errorf("unexpected field properties: %+v", userSchema.Fields[0])
	}

	if len(svc.Endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(svc.Endpoints))
	}

	for _, ep := range svc.Endpoints {
		switch h := ep.Handler.(type) {
		case service.OpenAPIHandler:
			if ep.Path == "/docs" {
				if h.Renderer != "scalar" {
					t.Errorf("expected renderer 'scalar', got %q", h.Renderer)
				}
				if h.SpecURL != "/openapi.json" {
					t.Errorf("expected auto-derived /openapi.json, got %q", h.SpecURL)
				}
			}
		case service.PipelineHandler:
			if ep.Path == "/api/v1/users" {
				if len(h.Steps) != 2 {
					t.Fatalf("expected 2 pipeline steps, got %d", len(h.Steps))
				}
				if h.Steps[0].SQL.ConnectionKey != "postgres.primary" && h.Steps[0].SQL.ConnectionKey != "connection.postgres.primary" {
					t.Errorf("unexpected sql connection key: %q", h.Steps[0].SQL.ConnectionKey)
				}
				if len(ep.RequestRules.HeaderFields) != 1 || len(ep.RequestRules.BodyFields) != 1 {
					t.Errorf("unexpected request rules: %+v", ep.RequestRules)
				}
			}
		}
	}
}

func TestBuild_Failures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		hcl         string
		expectError string
	}{
		{
			name: "rejects duplicate endpoint routes",
			hcl: `
endpoint "GET /records" {
  pipeline {
    respond { status = 200 }
  }
}
endpoint "GET /records" {
  pipeline {
    respond { status = 200 }
  }
}
`,
			expectError: `duplicate endpoint route "GET /records"`,
		},
		{
			name: "rejects conflicting handlers",
			hcl: `
endpoint "GET /conflict" {
  openapi "ui" {
    renderer = "scalar"
  }
  pipeline {
    respond { status = 200 }
  }
}
`,
			expectError: `endpoint "GET /conflict" defines conflicting handlers: 'openapi "ui"' and 'pipeline'`,
		},
		{
			name: "rejects multiple openapi blocks",
			hcl: `
endpoint "GET /multi" {
  openapi "spec" {
    format = "json"
  }
  openapi "ui" {
    renderer = "scalar"
  }
}
`,
			expectError: `endpoint "GET /multi" defines multiple openapi handlers ("spec", "ui")`,
		},
		{
			name: "rejects openapi endpoint with non-GET method",
			hcl: `
endpoint "POST /openapi.json" {
  openapi "spec" {
    format = "json"
  }
}
`,
			expectError: `endpoint "POST /openapi.json" is invalid; openapi endpoints only support HTTP GET and HEAD`,
		},
		{
			name: "rejects openapi endpoint with request block",
			hcl: `
endpoint "GET /docs" {
  request {
    query {
      field "q" { type = string }
    }
  }
  openapi "ui" {
    renderer = "scalar"
  }
}
`,
			expectError: `endpoint "GET /docs": openapi endpoints are engine-managed and do not accept a 'request' block`,
		},
		{
			name: "rejects unknown connection reference",
			hcl: `
endpoint "GET /items" {
  pipeline {
    sql "query_db" {
      connection = connection.postgres.non_existent
      query      = "SELECT 1"
    }
    respond { status = 200 }
  }
}
`,
			expectError: `unknown connection "connection.postgres.non_existent"`,
		},
		{
			name: "rejects unknown schema reference",
			hcl: `
endpoint "POST /items" {
  request {
    body = schema.missing_schema
  }
  pipeline {
    respond { status = 200 }
  }
}
`,
			expectError: `unknown schema reference "schema.missing_schema"`,
		},
		{
			name: "rejects invalid duration format in server block",
			hcl: `
server {
  read_timeout = "not-a-valid-duration"
}
endpoint "GET /ping" {
  pipeline {
    respond { status = 200 }
  }
}
`,
			expectError: "invalid duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := writeManifest(t, tt.hcl)
			_, err := service.Build(m, eval.BaseContext())
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectError)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected error %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

func TestBuild_NilManifest(t *testing.T) {
	t.Parallel()

	_, err := service.Build(nil, eval.BaseContext())
	if err == nil {
		t.Fatal("expected error building nil manifest, got nil")
	}
}
