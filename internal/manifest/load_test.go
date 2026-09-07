package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/manifest"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	return dir
}

func TestLoad_SingleFile(t *testing.T) {
	t.Parallel()

	hcl := `
server {
  host          = "0.0.0.0"
  port          = 9090
  read_timeout  = "45s"
  write_timeout = "45s"
  idle_timeout  = "120s"
  max_body_size = "15MB"
}

openapi {
  title       = "Store API"
  version     = "2.1.0"
  description = "E-Commerce backend API"
}

problem {
  type_prefix = "https://example.com/errors/"
}

connection "sqlite" "main" {
  source = "file:local.db?cache=shared"
  pool {
    max_open     = 10
    max_idle     = 2
    max_lifetime = "10m"
    idle_timeout = "2m"
  }
}

schema "user" {
  description = "User schema"
  field "email" {
    type     = string
    required = true
  }
}

endpoint "POST /users" {
  description = "Create user"
  request {
    headers {
      field "x-api-key" {
        type     = string
        required = true
      }
    }
    body = schema.user
  }
  pipeline {
    respond {
      status = 201
    }
  }
}

endpoint "GET /docs" {
  openapi "ui" {
    renderer = "scalar"
  }
}
`

	dir := writeFiles(t, map[string]string{"main.hcl": hcl})
	m, err := manifest.Load(dir, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if m.Server == nil || m.Server.Host != "0.0.0.0" || m.Server.Port != 9090 {
		t.Fatalf("unexpected server block: %+v", m.Server)
	}
	if m.Server.ReadTimeout != "45s" || m.Server.MaxBodySize != "15MB" {
		t.Errorf("expected raw string timeouts in manifest: %+v", m.Server)
	}

	if m.OpenAPI == nil || m.OpenAPI.Title != "Store API" || m.OpenAPI.Version != "2.1.0" {
		t.Fatalf("unexpected openapi block: %+v", m.OpenAPI)
	}

	if m.Problem == nil || m.Problem.TypePrefix != "https://example.com/errors/" {
		t.Fatalf("unexpected problem block: %+v", m.Problem)
	}

	if len(m.Connections) != 1 || m.Connections[0].Key() != "sqlite.main" {
		t.Fatalf("expected 1 connection sqlite.main, got: %+v", m.Connections)
	}
	if !strings.HasPrefix(m.Connections[0].Source, "file:") || !strings.Contains(m.Connections[0].Source, "local.db") {
		t.Errorf("expected resolved relative source, got %q", m.Connections[0].Source)
	}

	if len(m.Schemas) != 1 || m.Schemas[0].Name != "user" {
		t.Fatalf("expected 1 schema, got: %+v", m.Schemas)
	}

	if len(m.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(m.Endpoints))
	}

	postEp := m.Endpoints[0]
	if postEp.MethodAndPath != "POST /users" {
		t.Errorf("expected 'POST /users', got %q", postEp.MethodAndPath)
	}
	if postEp.Request == nil || postEp.Request.HeadersInline == nil || postEp.Request.BodyExpr == nil {
		t.Fatalf("expected decoded request block on endpoint: %+v", postEp.Request)
	}
	if len(postEp.Request.HeadersInline.Fields) != 1 {
		t.Errorf("expected 1 header field, got %d", len(postEp.Request.HeadersInline.Fields))
	}

	docsEp := m.Endpoints[1]
	if len(docsEp.OpenAPIs) != 1 || docsEp.OpenAPIs[0].Mode != "ui" {
		t.Fatalf("expected openapi ui block on docs endpoint: %+v", docsEp.OpenAPIs)
	}
}

func TestLoad_MultiFileDirectory(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"server.hcl": `
server {
  port = 8080
}
problem {
  type_prefix = "urn:api:error:"
}
`,
		"models/user.hcl": `
schema "user" {
  field "id" {
    type = int
  }
}
`,
		"routes/users.hcl": `
endpoint "GET /users" {
  pipeline {
    respond {
      status = 200
    }
  }
}
`,
	}

	dir := writeFiles(t, files)
	m, err := manifest.Load(dir, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected multi-file load error: %v", err)
	}

	if m.Server == nil || m.Server.Port != 8080 {
		t.Errorf("server port = %v; want 8080", m.Server)
	}
	if m.Problem == nil || m.Problem.TypePrefix != "urn:api:error:" {
		t.Errorf("problem prefix = %v; want urn:api:error:", m.Problem)
	}
	if len(m.Schemas) != 1 || m.Schemas[0].Name != "user" {
		t.Errorf("expected 1 schema 'user', got %+v", m.Schemas)
	}
	if len(m.Endpoints) != 1 || m.Endpoints[0].MethodAndPath != "GET /users" {
		t.Errorf("expected 1 endpoint 'GET /users', got %+v", m.Endpoints)
	}
}

func TestLoad_DuplicateSingletons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		files       map[string]string
		expectError string
	}{
		{
			name: "duplicate server block across files",
			files: map[string]string{
				"a.hcl": `server { port = 8080 }`,
				"b.hcl": `server { port = 9000 }`,
			},
			expectError: "duplicate singleton block 'server'",
		},
		{
			name: "duplicate openapi block across files",
			files: map[string]string{
				"a.hcl": `openapi { title = "API A" }`,
				"b.hcl": `openapi { title = "API B" }`,
			},
			expectError: "duplicate singleton block 'openapi'",
		},
		{
			name: "duplicate problem block across files",
			files: map[string]string{
				"a.hcl": `problem { type_prefix = "https://a.com/" }`,
				"b.hcl": `problem { type_prefix = "https://b.com/" }`,
			},
			expectError: "duplicate singleton block 'problem'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := writeFiles(t, tt.files)
			_, err := manifest.Load(dir, eval.BaseContext())
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectError)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected error %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

func TestLoad_IgnoresHiddenAndNonHCL(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"main.hcl": `
schema "item" {
  field "title" {
    type = string
  }
}
`,
		".git/config.hcl": `
server {
  port = 6666
}
`,
		"notes.txt":   `this is not hcl`,
		"schema.json": `{"type": "object"}`,
	}

	dir := writeFiles(t, files)
	m, err := manifest.Load(dir, eval.BaseContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Server != nil {
		t.Errorf("expected .git/config.hcl to be ignored, but server block was loaded")
	}
	if len(m.Schemas) != 1 || m.Schemas[0].Name != "item" {
		t.Errorf("expected 1 schema 'item', got %+v", m.Schemas)
	}
}

func TestLoad_SyntaxError(t *testing.T) {
	t.Parallel()

	dir := writeFiles(t, map[string]string{
		"broken.hcl": `endpoint "GET /test" { invalid syntax here }`,
	})

	_, err := manifest.Load(dir, eval.BaseContext())
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

func TestLoad_NonExistentPath(t *testing.T) {
	t.Parallel()

	_, err := manifest.Load("/path/that/does/not/exist_12345", eval.BaseContext())
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}

func TestLoad_ResolveRelativePath(t *testing.T) {
	t.Parallel()

	baseDir := filepath.Join(string(filepath.Separator), "app", "configs")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sqlite memory special case",
			input:    ":memory:",
			expected: ":memory:",
		},
		{
			name:     "sqlite file memory uri",
			input:    "file::memory:?cache=shared",
			expected: "file::memory:?cache=shared",
		},
		{
			name:     "remote database dsn",
			input:    "postgres://user:pass@localhost:5432/db",
			expected: "postgres://user:pass@localhost:5432/db",
		},
		{
			name:     "relative sqlite file uri",
			input:    "file:data/store.db?cache=shared",
			expected: "file:" + filepath.ToSlash(filepath.Join(baseDir, "data", "store.db")) + "?cache=shared",
		},
		{
			name:     "relative file path",
			input:    "data/store.db",
			expected: filepath.Join(baseDir, "data", "store.db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := writeFiles(t, map[string]string{
				"main.hcl": `
connection "sqlite" "test" {
  source = "` + tt.input + `"
}
`,
			})

			m, err := manifest.Load(dir, eval.BaseContext())
			if err != nil {
				t.Fatalf("unexpected load error: %v", err)
			}

			if len(m.Connections) != 1 {
				t.Fatalf("expected 1 connection, got %d", len(m.Connections))
			}

			if tt.input == ":memory:" || tt.input == "file::memory:?cache=shared" || strings.Contains(tt.input, "://") {
				if m.Connections[0].Source != tt.expected {
					t.Errorf("got source %q; want %q", m.Connections[0].Source, tt.expected)
				}
			} else {
				if !filepath.IsAbs(strings.TrimPrefix(m.Connections[0].Source, "file:")) {
					t.Errorf("expected absolute resolved path, got %q", m.Connections[0].Source)
				}
			}
		})
	}
}
