package xsql_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/connections/xsql"
	"github.com/ju4n97/hclapi/internal/core"
)

// mockDriver implements database/sql/driver for unit testing without network dependencies.
type mockDriver struct{}

func (m *mockDriver) Open(name string) (driver.Conn, error) {
	if strings.Contains(name, "fail_ping") {
		return &mockConn{failPing: true}, nil
	}
	return &mockConn{}, nil
}

type mockConn struct {
	failPing bool
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c *mockConn) Close() error                              { return nil }
func (c *mockConn) Begin() (driver.Tx, error)                 { return nil, nil }
func (c *mockConn) Ping(ctx context.Context) error {
	if c.failPing {
		return sql.ErrConnDone
	}
	return nil
}

var registerMockDriverOnce sync.Once

func ensureMockDriver() {
	registerMockDriverOnce.Do(func() {
		sql.Register("mock_sql", &mockDriver{})
	})
}

func TestDialects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		driver      string
		expected    string
		placeholder string
	}{
		{"postgres", "postgres", "$1"},
		{"cockroachdb", "postgres", "$1"},
		{"sqlite", "sqlite", "?"},
		{"mysql", "mysql", "?"},
		{"clickhouse", "mysql", "?"},
		{"snowflake", "mysql", "?"},
		{"duckdb", "mysql", "?"},
		{"sqlserver", "sqlserver", "@p1"},
		{"oracle", "oracle", ":1"},
	}

	for _, tt := range tests {
		d := xsql.ResolveDialect(tt.driver)
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

	ensureMockDriver()

	t.Run("Registers and retrieves database pool", func(t *testing.T) {
		t.Parallel()

		mgr := xsql.NewManager()
		conn := core.Connection{
			Driver: "mock_sql",
			Name:   "primary",
			URL:    "mock://valid",
			Pool: core.PoolConfig{
				MaxOpenConns:    10,
				MaxIdleConns:    2,
				ConnMaxLifetime: core.Duration(15 * time.Minute),
				IdleTimeout:     core.Duration(5 * time.Minute),
			},
		}

		err := mgr.Open(t.Context(), conn)
		if err != nil {
			t.Fatalf("unexpected error opening pool: %v", err)
		}

		// Retrieve via short key
		pool, exists := mgr.Get("mock_sql.primary")
		if !exists || pool == nil {
			t.Fatalf("expected pool to exist for mock_sql.primary")
		}

		// Retrieve via full HCL reference
		poolRef, existsRef := mgr.Get("connection.mock_sql.primary")
		if !existsRef || poolRef == nil {
			t.Fatalf("expected pool to exist for connection.mock_sql.primary")
		}

		// Close manager
		if err := mgr.Close(); err != nil {
			t.Fatalf("unexpected error closing manager: %v", err)
		}

		// Verify empty after close
		if _, existsAfter := mgr.Get("mock_sql.primary"); existsAfter {
			t.Errorf("expected pool to be removed after close")
		}
	})

	t.Run("Fails fast on ping error", func(t *testing.T) {
		t.Parallel()

		mgr := xsql.NewManager()
		conn := core.Connection{
			Driver: "mock_sql",
			Name:   "broken",
			URL:    "mock://fail_ping",
			Pool:   core.DefaultPoolConfig(),
		}

		err := mgr.Open(t.Context(), conn)
		if err == nil {
			t.Fatal("expected error on failed ping, got nil")
		}
	})
}
