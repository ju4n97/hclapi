package xsql_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/steps/xsql"
)

func TestRewriteNamedQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		driver       string
		query        string
		args         map[string]any
		expectedSQL  string
		expectedArgs []any
	}{
		{
			name:         "Postgres placeholders ($1, $2)",
			driver:       "postgres",
			query:        "SELECT * FROM users WHERE id = @id AND email = @email",
			args:         map[string]any{"id": "usr_1", "email": "a@b.com"},
			expectedSQL:  "SELECT * FROM users WHERE id = $1 AND email = $2",
			expectedArgs: []any{"usr_1", "a@b.com"},
		},
		{
			name:         "SQLite positional placeholders (?)",
			driver:       "sqlite",
			query:        "UPDATE accounts SET status = @status WHERE id = @id",
			args:         map[string]any{"status": "active", "id": 42},
			expectedSQL:  "UPDATE accounts SET status = ? WHERE id = ?",
			expectedArgs: []any{"active", 42},
		},
		{
			name:         "SQLServer placeholders (@p1, @p2)",
			driver:       "sqlserver",
			query:        "SELECT name FROM products WHERE sku = @sku",
			args:         map[string]any{"sku": "SKU-99"},
			expectedSQL:  "SELECT name FROM products WHERE sku = @p1",
			expectedArgs: []any{"SKU-99"},
		},
		{
			name:         "Oracle placeholders (:1, :2)",
			driver:       "oracle",
			query:        "INSERT INTO logs (action, user_id) VALUES (@action, @id)",
			args:         map[string]any{"action": "LOGIN", "id": 100},
			expectedSQL:  "INSERT INTO logs (action, user_id) VALUES (:1, :2)",
			expectedArgs: []any{"LOGIN", 100},
		},
		{
			name:         "Handles nil args map gracefully",
			driver:       "sqlite",
			query:        "SELECT * FROM items WHERE status = @status",
			args:         nil,
			expectedSQL:  "SELECT * FROM items WHERE status = ?",
			expectedArgs: []any{nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := connsql.ResolveDialect(tt.driver)
			rewritten, args, err := xsql.RewriteNamedQuery(tt.query, tt.args, d)
			if err != nil {
				t.Fatalf("unexpected rewrite error: %v", err)
			}
			if rewritten != tt.expectedSQL {
				t.Errorf("expected SQL %q, got %q", tt.expectedSQL, rewritten)
			}
			if len(args) != len(tt.expectedArgs) {
				t.Fatalf("expected %d args, got %d", len(tt.expectedArgs), len(args))
			}
			for i := range args {
				if args[i] != tt.expectedArgs[i] {
					t.Errorf("arg %d: expected %v, got %v", i, tt.expectedArgs[i], args[i])
				}
			}
		})
	}
}

func setupTestSQLiteDB(t *testing.T) *connsql.Pool {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			raw_bytes BLOB,
			created_at TEXT,
			is_active INTEGER
		);
		INSERT INTO users VALUES (1, 'Jane', X'627974655f737472696e67', '2026-08-30T12:00:00Z', 1);
		INSERT INTO users VALUES (2, 'John', X'7365636f6e64', '2026-08-30T13:00:00Z', 0);
	`
	if _, err := db.ExecContext(t.Context(), schema); err != nil {
		t.Fatalf("failed to seed test db: %v", err)
	}

	return &connsql.Pool{
		DB:      db,
		Dialect: connsql.ResolveDialect("sqlite"),
		Config: core.Connection{
			Driver: "sqlite",
			Name:   "test",
			URL:    "file::memory:?cache=shared",
		},
	}
}

func TestExecute(t *testing.T) {
	t.Parallel()

	pool := setupTestSQLiteDB(t)

	t.Run("Executes SELECT query with row scanning and type normalization", func(t *testing.T) {
		t.Parallel()

		res, err := xsql.Execute(
			t.Context(),
			pool,
			"SELECT id, name, raw_bytes, created_at FROM users WHERE id = @id",
			map[string]any{"id": 1},
		)
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		if res.RowsAffected != 1 {
			t.Errorf("expected 1 row affected, got %d", res.RowsAffected)
		}
		if len(res.Rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(res.Rows))
		}

		if res.Row["name"] != "Jane" {
			t.Errorf("expected name 'Jane', got %v", res.Row["name"])
		}

		// Verify BLOB normalized to string
		if rawBytes, ok := res.Row["raw_bytes"].(string); !ok || rawBytes != "byte_string" {
			t.Errorf(
				"expected raw_bytes to be normalized to string 'byte_string', got %T (%v)",
				res.Row["raw_bytes"],
				res.Row["raw_bytes"],
			)
		}
	})

	t.Run("Executes non-returning mutation with rows_affected count", func(t *testing.T) {
		t.Parallel()

		res, err := xsql.Execute(
			t.Context(),
			pool,
			"UPDATE users SET is_active = 1 WHERE is_active = @status",
			map[string]any{"status": 0},
		)
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		if res.RowsAffected != 1 {
			t.Errorf("expected 1 row updated, got %d", res.RowsAffected)
		}
		if len(res.Rows) != 0 {
			t.Errorf("expected empty rows slice for mutation, got %d", len(res.Rows))
		}
		if res.Row != nil {
			t.Errorf("expected nil row for mutation, got %+v", res.Row)
		}
	})

	t.Run("Returns empty slice and nil row when 0 rows match", func(t *testing.T) {
		t.Parallel()

		res, err := xsql.Execute(t.Context(), pool, "SELECT * FROM users WHERE id = @id", map[string]any{"id": 9999})
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		if res.RowsAffected != 0 {
			t.Errorf("expected 0 rows affected, got %d", res.RowsAffected)
		}
		if len(res.Rows) != 0 {
			t.Errorf("expected empty rows slice, got %d", len(res.Rows))
		}
		if res.Row != nil {
			t.Errorf("expected nil row, got %+v", res.Row)
		}
	})
}
