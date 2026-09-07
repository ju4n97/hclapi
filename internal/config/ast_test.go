package config_test

import (
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/config"
)

func TestServer_SetDefaults(t *testing.T) {
	t.Parallel()

	var s config.Server
	s.SetDefaults()

	if s.Host != "127.0.0.1" {
		t.Errorf("Host = %q; want 127.0.0.1", s.Host)
	}
	if s.Port != 8080 {
		t.Errorf("Port = %d; want 8080", s.Port)
	}
	if s.ReadTimeout.Duration() != 15*time.Second {
		t.Errorf("ReadTimeout = %v; want 15s", s.ReadTimeout)
	}
	if s.MaxBodySize.Bytes() != 10*1024*1024 {
		t.Errorf("MaxBodySize = %d; want 10MB", s.MaxBodySize.Bytes())
	}
}

func TestProblem_FormatType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefix string
		slug   string
		want   string
	}{
		{"", "not-found", "urn:hclapi:error:not-found"},
		{"https://docs.example.com/errors/", "bad-request", "https://docs.example.com/errors/bad-request"},
		{"https://docs.example.com/errors", "conflict", "https://docs.example.com/errors/conflict"},
		{"urn:acme:error:", "unauthorized", "urn:acme:error:unauthorized"},
	}

	for _, tt := range tests {
		p := config.Problem{TypePrefix: tt.prefix}
		if got := p.FormatType(tt.slug); got != tt.want {
			t.Errorf("FormatType(%q) with prefix %q = %q; want %q", tt.slug, tt.prefix, got, tt.want)
		}
	}
}

func TestOpenAPI_SetDefaults(t *testing.T) {
	t.Parallel()

	var o config.OpenAPI
	o.SetDefaults()

	if o.Title != "API Documentation" {
		t.Errorf("Title = %q; want 'API Documentation'", o.Title)
	}
	if o.Version != "1.0.0" {
		t.Errorf("Version = %q; want '1.0.0'", o.Version)
	}
}

func TestConnection_KeysAndDefaults(t *testing.T) {
	t.Parallel()

	c := config.Connection{
		Driver: "postgres",
		Name:   "primary",
		Source: "postgres://localhost/db",
	}
	c.SetDefaults()

	if c.Key() != "postgres.primary" {
		t.Errorf("Key() = %q; want 'postgres.primary'", c.Key())
	}
	if c.Reference() != "connection.postgres.primary" {
		t.Errorf("Reference() = %q; want 'connection.postgres.primary'", c.Reference())
	}
	if c.Pool.MaxOpen != 25 {
		t.Errorf("Pool.MaxOpen = %d; want 25", c.Pool.MaxOpen)
	}
	if c.Pool.MaxLifetime.Duration() != 30*time.Minute {
		t.Errorf("Pool.MaxLifetime = %v; want 30m", c.Pool.MaxLifetime)
	}
}

func TestEndpoint_SealedHandler(t *testing.T) {
	t.Parallel()

	pipeEp := config.Endpoint{
		Method:  "POST",
		Path:    "/users",
		Handler: config.PipelineHandler{Steps: []config.ParsedStep{{Type: config.StepTypeRespond}}},
	}

	openEp := config.Endpoint{
		Method:  "GET",
		Path:    "/docs",
		Handler: config.OpenAPIHandler{Mode: "ui", Renderer: "scalar"},
	}

	switch h := pipeEp.Handler.(type) {
	case config.PipelineHandler:
		if len(h.Steps) != 1 {
			t.Errorf("expected 1 step, got %d", len(h.Steps))
		}
	default:
		t.Fatalf("expected PipelineHandler, got %T", h)
	}

	switch h := openEp.Handler.(type) {
	case config.OpenAPIHandler:
		if h.Renderer != "scalar" {
			t.Errorf("expected 'scalar', got %q", h.Renderer)
		}
	default:
		t.Fatalf("expected OpenAPIHandler, got %T", h)
	}
}
