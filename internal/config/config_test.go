package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/config"
	"github.com/ju4n97/hclapi/internal/eval"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "main.hcl")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return tmpDir
}

func TestConfig_Load_Success(t *testing.T) {
	t.Parallel()

	hcl := `
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
`
	dir := writeConfig(t, hcl)
	cfg, err := config.Load(dir, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	// Server assertions
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != 9000 {
		t.Errorf("unexpected server: %+v", cfg.Server)
	}
	if cfg.Server.ReadTimeout.Duration() != 30*time.Second {
		t.Errorf("expected 30s read_timeout, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.MaxBodySize.Bytes() != 25*1000*1000 {
		t.Errorf("expected 25MB max_body_size, got %d", cfg.Server.MaxBodySize.Bytes())
	}

	// Problem assertions
	if cfg.Problem.TypePrefix != "https://docs.example.com/errors/" {
		t.Errorf("unexpected problem prefix: %q", cfg.Problem.TypePrefix)
	}

	// Connection assertions
	if len(cfg.Connections) != 1 || cfg.Connections[0].Key() != "postgres.primary" {
		t.Fatalf("expected 1 connection 'postgres.primary', got: %+v", cfg.Connections)
	}
	if cfg.Connections[0].Pool.MaxOpen != 50 {
		t.Errorf("expected max_open 50, got %d", cfg.Connections[0].Pool.MaxOpen)
	}

	// Schema assertions
	userSchema, ok := cfg.Schemas["user_create"]
	if !ok || len(userSchema.Fields) != 1 {
		t.Fatalf("expected schema 'user_create', got: %+v", cfg.Schemas)
	}
	if userSchema.Fields[0].Type != "string" || !userSchema.Fields[0].Required {
		t.Errorf("unexpected field: %+v", userSchema.Fields[0])
	}

	// Endpoint assertions
	if len(cfg.Endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(cfg.Endpoints))
	}

	for _, ep := range cfg.Endpoints {
		switch h := ep.Handler.(type) {
		case config.OpenAPIHandler:
			if ep.Path == "/docs" {
				if h.Renderer != "scalar" {
					t.Errorf("expected scalar renderer, got %q", h.Renderer)
				}
				if h.SpecURL != "/openapi.json" {
					t.Errorf("expected auto-derived spec_url '/openapi.json', got %q", h.SpecURL)
				}
			}
		case config.PipelineHandler:
			if ep.Path == "/api/v1/users" {
				if len(h.Steps) != 2 {
					t.Fatalf("expected 2 steps, got %d", len(h.Steps))
				}
				if h.Steps[0].Name != "insert_user" || h.Steps[0].Type != config.StepTypeSQL {
					t.Errorf("unexpected step 0: %+v", h.Steps[0])
				}
			}
		default:
			t.Fatalf("unexpected handler type for endpoint: %+v", ep)
		}
	}
}

func TestConfig_ValidationFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		hcl         string
		expectError string
	}{
		{
			name: "Rejects duplicate endpoint routes",
			hcl: `
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
			name: "Rejects endpoint with both pipeline and openapi blocks",
			hcl: `
endpoint "GET /api/v1/invalid" {
  openapi "ui" {
    renderer = "scalar"
  }
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
			expectError: `endpoint "GET /api/v1/invalid" defines conflicting handlers: 'openapi "ui"' and 'pipeline'`,
		},
		{
			name: "Rejects endpoint with multiple openapi blocks",
			hcl: `
endpoint "GET /api/v1/multi-openapi" {
  openapi "spec" {
    format = "json"
  }
  openapi "ui" {
    renderer = "scalar"
  }
}
`,
			expectError: `endpoint "GET /api/v1/multi-openapi" defines multiple openapi handlers ("spec", "ui")`,
		},
		{
			name: "Rejects openapi endpoint with non-GET/HEAD method",
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
			name: "Rejects openapi endpoint with request block",
			hcl: `
endpoint "GET /docs" {
  request {
    query {
      field "q" {
        type = string
      }
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
			name: "Rejects unknown SQL connection reference",
			hcl: `
endpoint "GET /records" {
  pipeline {
    sql "fetch" {
      connection = connection.postgres.non_existent
      query      = "SELECT 1"
    }
    respond {
      status = 200
    }
  }
}
`,
			expectError: `unknown connection "connection.postgres.non_existent"`,
		},
		{
			name: "Rejects unknown schema reference",
			hcl: `
endpoint "POST /records" {
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
			expectError: `unknown schema reference "schema.missing_schema"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := writeConfig(t, tt.hcl)
			_, err := config.Load(dir, eval.BaseContext())
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectError)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected error %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}
