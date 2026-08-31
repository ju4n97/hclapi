package connsql_test

import (
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/core"
)

func TestDialects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		driver      string
		expected    string
		placeholder string
	}{
		{"postgres", "postgres", "$1"},
		{"cockroachdb", "cockroachdb", "$1"},
		{"sqlite", "sqlite", "?"},
		{"mysql", "mysql", "?"},
		{"clickhouse", "clickhouse", "?"},
		{"duckdb", "duckdb", "?"},
		{"sqlserver", "sqlserver", "@p1"},
		{"oracle", "oracle", ":1"},
	}

	for _, tt := range tests {
		d := connsql.ResolveDialect(tt.driver)
		if d.Name() != tt.expected {
			t.Errorf("for driver %q expected dialect %q, got %q", tt.driver, tt.expected, d.Name())
		}
		if p := d.Placeholder(0, "id"); p != tt.placeholder {
			t.Errorf("for driver %q expected placeholder %q, got %q", tt.driver, tt.placeholder, p)
		}
	}
}

func TestManager(t *testing.T) {
	t.Parallel()

	t.Run("Registers, configures pool, and retrieves sqlite database", func(t *testing.T) {
		t.Parallel()

		mgr := connsql.NewManager()
		conn := core.Connection{
			Driver: "sqlite",
			Name:   "primary",
			URL:    "file::memory:?cache=shared",
			Pool: core.PoolConfig{
				MaxOpenConns:    10,
				MaxIdleConns:    2,
				ConnMaxLifetime: core.Duration(15 * time.Minute),
				IdleTimeout:     core.Duration(5 * time.Minute),
			},
		}

		if err := mgr.Open(t.Context(), conn); err != nil {
			t.Fatalf("unexpected error opening pool: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })

		// Retrieve via short key
		pool, exists := mgr.Get("sqlite.primary")
		if !exists || pool == nil {
			t.Fatalf("expected pool to exist for sqlite.primary")
		}

		// Retrieve via full HCL reference
		poolRef, existsRef := mgr.Get("connection.sqlite.primary")
		if !existsRef || poolRef == nil {
			t.Fatalf("expected pool to exist for connection.sqlite.primary")
		}

		// Close manager
		if err := mgr.Close(); err != nil {
			t.Fatalf("unexpected error closing manager: %v", err)
		}

		// Verify empty after close
		if _, existsAfter := mgr.Get("sqlite.primary"); existsAfter {
			t.Errorf("expected pool to be removed after close")
		}
	})

	t.Run("Fails fast on invalid connection URL", func(t *testing.T) {
		t.Parallel()

		mgr := connsql.NewManager()
		conn := core.Connection{
			Driver: "sqlite",
			Name:   "broken",
			URL:    "file:/invalid_non_existent_dir_9999/app.db?mode=ro",
			Pool:   core.DefaultPoolConfig(),
		}

		if err := mgr.Open(t.Context(), conn); err == nil {
			t.Fatal("expected error on invalid connection URL, got nil")
		}
	})
}
