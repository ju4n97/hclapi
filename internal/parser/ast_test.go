package parser_test

import (
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
)

func parseHCL(t *testing.T, src string) hcl.Expression {
	t.Helper()

	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("syntax error in expression %q: %s", src, diags.Error())
	}

	return expr
}

func TestServerBlock_ToServer(t *testing.T) {
	t.Parallel()

	t.Run("Applies custom durations, byte sizes, and defaults", func(t *testing.T) {
		t.Parallel()

		block := &parser.ServerBlock{
			Host:         "0.0.0.0",
			Port:         3000,
			ReadTimeout:  new("30s"),
			WriteTimeout: new("45s"),
			IdleTimeout:  new("2m"),
			MaxBodySize:  new("50MB"),
		}

		srv, err := block.ToServer()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if srv.Host != "0.0.0.0" || srv.Port != 3000 {
			t.Errorf("unexpected host/port: %+v", srv)
		}
		if srv.ReadTimeout.Duration() != 30*time.Second {
			t.Errorf("expected read timeout 30s, got %v", srv.ReadTimeout)
		}
		if srv.MaxBodySize.Bytes() != 50*1000*1000 {
			t.Errorf("expected max body size 50MB, got %d", srv.MaxBodySize.Bytes())
		}
	})

	t.Run("Rejects malformed read_timeout duration", func(t *testing.T) {
		t.Parallel()

		block := &parser.ServerBlock{
			ReadTimeout: new("100years"),
		}
		_, err := block.ToServer()
		if err == nil {
			t.Fatal("expected error for invalid duration, got nil")
		}
	})

	t.Run("Rejects malformed max_body_size unit", func(t *testing.T) {
		t.Parallel()

		block := &parser.ServerBlock{
			MaxBodySize: new("10XB"),
		}
		_, err := block.ToServer()
		if err == nil {
			t.Fatal("expected error for invalid byte size, got nil")
		}
	})
}

func TestProblemBlock_ToProblem(t *testing.T) {
	t.Parallel()

	t.Run("Maps prefix cleanly", func(t *testing.T) {
		t.Parallel()

		block := &parser.ProblemBlock{
			TypePrefix: new("https://docs.example.com/errors/"),
		}

		p := block.ToProblem()
		if p.TypePrefix != "https://docs.example.com/errors/" {
			t.Errorf("expected prefix, got %q", p.TypePrefix)
		}
		if p.ProblemType("not-found") != "https://docs.example.com/errors/not-found" {
			t.Errorf("unexpected ProblemType: %q", p.ProblemType("not-found"))
		}
	})

	t.Run("Handles nil block", func(t *testing.T) {
		t.Parallel()

		var block *parser.ProblemBlock
		p := block.ToProblem()
		if p.TypePrefix != "" {
			t.Errorf("expected empty prefix, got %q", p.TypePrefix)
		}
	})
}

func TestOpenAPIBlock_ToOpenAPI(t *testing.T) {
	t.Parallel()

	t.Run("Maps metadata and evaluable servers and tags", func(t *testing.T) {
		t.Parallel()

		block := &parser.OpenAPIBlock{
			Title:       new("Acme API"),
			Version:     new("2.0.0"),
			Description: new("API docs"),
			Servers:     parseHCL(t, `[{ url = "https://api.example.com", description = "prod" }]`),
			Tags:        parseHCL(t, `[{ name = "users", description = "User endpoints" }]`),
			Contact: &parser.OpenAPIContactBlock{
				Name: new("Support"),
			},
			License: &parser.OpenAPILicenseBlock{
				Name: new("Apache-2.0"),
			},
		}

		cfg, err := block.ToOpenAPI(eval.BaseContext())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Title != "Acme API" || cfg.Version != "2.0.0" {
			t.Errorf("unexpected title/version: %+v", cfg)
		}
		if len(cfg.Servers) != 1 || cfg.Servers[0].URL != "https://api.example.com" {
			t.Errorf("unexpected servers: %+v", cfg.Servers)
		}
		if len(cfg.Tags) != 1 || cfg.Tags[0].Name != "users" {
			t.Errorf("unexpected tags: %+v", cfg.Tags)
		}
		if cfg.Contact == nil || cfg.Contact.Name != "Support" {
			t.Errorf("unexpected contact: %+v", cfg.Contact)
		}
		if cfg.License == nil || cfg.License.Name != "Apache-2.0" {
			t.Errorf("unexpected license: %+v", cfg.License)
		}
	})
}

func TestConnectionBlock_ToConnection(t *testing.T) {
	t.Parallel()

	t.Run("Applies custom pool settings and resolves relative path", func(t *testing.T) {
		t.Parallel()

		block := &parser.ConnectionBlock{
			Driver: "postgres",
			Name:   "primary",
			Source: "postgres://user:pass@localhost:5432/db",
			Pool: &parser.ConnectionPoolBlock{
				MaxOpen:     new(50),
				MaxIdle:     new(10),
				MaxLifetime: new("1h"),
				IdleTimeout: new("10m"),
			},
		}

		conn, err := block.ToConnection()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.Driver != "postgres" || conn.Name != "primary" {
			t.Errorf("unexpected driver/name: %s.%s", conn.Driver, conn.Name)
		}
		if conn.Pool.MaxOpen != 50 || conn.Pool.MaxIdle != 10 {
			t.Errorf("unexpected pool limits: %+v", conn.Pool)
		}
		if conn.Pool.MaxLifetime.Duration() != time.Hour {
			t.Errorf("expected 1h lifetime, got %v", conn.Pool.MaxLifetime)
		}
	})

	t.Run("Rejects malformed pool duration", func(t *testing.T) {
		t.Parallel()

		block := &parser.ConnectionBlock{
			Driver: "postgres",
			Name:   "primary",
			Source: "postgres://localhost/db",
			Pool: &parser.ConnectionPoolBlock{
				MaxLifetime: new("bad_duration"),
			},
		}

		_, err := block.ToConnection()
		if err == nil {
			t.Fatal("expected error for invalid duration, got nil")
		}
	})
}

func TestFieldBlock_ToField(t *testing.T) {
	t.Parallel()

	evalCtx := eval.BaseContext()

	t.Run("Compiles field types, constraints, and defaults", func(t *testing.T) {
		t.Parallel()

		block := parser.FieldBlock{
			Name:        "email",
			Type:        parseHCL(t, "string"),
			Required:    true,
			Format:      new("email"),
			Description: new("User email"),
		}

		field, err := block.ToField(evalCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if field.Name != "email" || field.Type != "string" || !field.Required {
			t.Errorf("unexpected field compilation: %+v", field)
		}
		if field.Format != "email" || field.Description != "User email" {
			t.Errorf("unexpected format/description: %+v", field)
		}
	})

	t.Run("Compiles enum list and list type constructor", func(t *testing.T) {
		t.Parallel()

		block := parser.FieldBlock{
			Name:        "tags",
			Type:        parseHCL(t, "list(string)"),
			Enum:        parseHCL(t, `["go", "api"]`),
			MinItems:    new(1),
			UniqueItems: true,
		}

		field, err := block.ToField(evalCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if field.Type != "list(string)" {
			t.Errorf("expected type 'list(string)', got %q", field.Type)
		}
		if len(field.Enum) != 2 || field.Enum[0] != "go" {
			t.Errorf("unexpected enum: %+v", field.Enum)
		}
		if !field.UniqueItems || *field.MinItems != 1 {
			t.Errorf("unexpected items constraints: %+v", field)
		}
	})
}
