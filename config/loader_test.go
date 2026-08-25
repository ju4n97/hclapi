package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ekisa-team/sqlmux/config"
)

func TestLoad_ValidConfigs(t *testing.T) {
	t.Parallel()

	paths := []string{
		"testdata/valid/minimal.yaml",
		"testdata/valid/multi_resource.yaml",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.Load(path)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected non-nil config")
			}
		})
	}
}

func TestLoad_InvalidConfigs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		wantErrSubstr string
	}{
		{"bad duration string", "testdata/invalid/bad_duration.yaml", ""},
		{"bad field type enum", "testdata/invalid/bad_field_type.yaml", "invalid config"},
		{"bad method enum", "testdata/invalid/bad_method.yaml", "invalid config"},
		{"bad param source pattern", "testdata/invalid/bad_param_source.yaml", "invalid config"},
		{"malformed yaml", "testdata/invalid/malformed.yaml", "failed to parse config file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.Load(tt.path)
			if err == nil {
				t.Fatalf("expected error, got none (config: %+v)", cfg)
			}
			if cfg != nil {
				t.Fatalf("expected nil config on error, got: %+v", cfg)
			}
			if tt.wantErrSubstr != "" && !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("expected error to contain %q, got: %v", tt.wantErrSubstr, err)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	if _, err := config.Load("testdata/does_not_exist.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	t.Parallel()

	if _, err := config.Load("testdata/invalid/malformed.yaml"); err == nil {
		t.Fatal("expected error for malformed yaml")
	}
}

func TestLoad_FieldMapping(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("testdata/valid/multi_resource.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("port: got %d, want 8080", cfg.Server.Port)
	}
	if got := cfg.Server.ReadTimeout.Duration(); got != 10*time.Second {
		t.Errorf("read_timeout: got %v, want 10s", got)
	}
	if got := cfg.Database.Pool.ConnMaxLifetime.Duration(); got != 30*time.Minute {
		t.Errorf("conn_max_lifetime: got %v, want 30m", got)
	}

	if len(cfg.API.Resources) != 2 {
		t.Fatalf("resources: got %d, want 2", len(cfg.API.Resources))
	}

	patients := cfg.API.Resources[0]
	if patients.Name != "patients" || len(patients.Routes) != 4 {
		t.Fatalf("unexpected patients resource: %+v", patients)
	}

	getByID := patients.Routes[2] // GET /:id via procedure
	if getByID.Query.Procedure != "dbo.GetPatient" {
		t.Errorf("procedure: got %q, want dbo.GetPatient", getByID.Query.Procedure)
	}
	if getByID.Query.Params["id"] != "path_params.id" {
		t.Errorf("params.id: got %q, want path_params.id", getByID.Query.Params["id"])
	}

	appointments := cfg.API.Resources[1]
	create := appointments.Routes[0]
	if create.Body["scheduled_at"].Type != "datetime" {
		t.Errorf("scheduled_at type: got %q, want datetime", create.Body["scheduled_at"].Type)
	}
}

func TestLoad_OptionalBodyOnListRoute(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	route := cfg.API.Resources[0].Routes[0]
	if route.Body != nil {
		t.Errorf("expected nil body on GET-list route, got: %+v", route.Body)
	}
}
