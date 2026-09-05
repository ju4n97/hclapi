package xsql_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/steps/xsql"
)

// setupIsolatedSQLitePool creates a private in-memory SQLite database unique to each subtest.
func setupIsolatedSQLitePool(t *testing.T, dbName string) *connsql.Pool {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			raw_bytes BLOB,
			created_at TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1
		);
		INSERT INTO users VALUES (1, 'Jane Developer', 'jane@example.com', X'627974655f64617461', '2026-08-30T12:00:00Z', 1);
		INSERT INTO users VALUES (2, 'John Doe', 'john@example.com', X'7365636f6e64', '2026-08-30T13:00:00Z', 0);
	`
	if _, err := db.ExecContext(t.Context(), schema); err != nil {
		t.Fatalf("failed to seed isolated test db: %v", err)
	}

	return &connsql.Pool{
		DB:      db,
		Dialect: connsql.ResolveDialect("sqlite"),
		Config: manifest.Connection{
			Driver: "sqlite",
			Name:   dbName,
			Source: dsn,
		},
	}
}

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
			args:         map[string]any{"id": 1, "email": "jane@example.com"},
			expectedSQL:  "SELECT * FROM users WHERE id = $1 AND email = $2",
			expectedArgs: []any{1, "jane@example.com"},
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
			name:         "SQL Server placeholders (@p1, @p2)",
			driver:       "sqlserver",
			query:        "SELECT name FROM products WHERE sku = @sku AND active = @active",
			args:         map[string]any{"sku": "SKU-99", "active": true},
			expectedSQL:  "SELECT name FROM products WHERE sku = @p1 AND active = @p2",
			expectedArgs: []any{"SKU-99", true},
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
			name:         "Repeated named parameter in query",
			driver:       "postgres",
			query:        "SELECT * FROM users WHERE email = @email OR backup_email = @email",
			args:         map[string]any{"email": "jane@example.com"},
			expectedSQL:  "SELECT * FROM users WHERE email = $1 OR backup_email = $2",
			expectedArgs: []any{"jane@example.com", "jane@example.com"},
		},
		{
			name:         "Missing argument in map maps to nil",
			driver:       "sqlite",
			query:        "SELECT * FROM items WHERE status = @missing_arg",
			args:         map[string]any{},
			expectedSQL:  "SELECT * FROM items WHERE status = ?",
			expectedArgs: []any{nil},
		},
		{
			name:         "Nil args map maps all placeholders to nil",
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

func TestExecute(t *testing.T) {
	t.Parallel()

	t.Run("Executes SELECT query with row scanning and type normalization", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_select")

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
		if res.Row["name"] != "Jane Developer" {
			t.Errorf("expected name 'Jane Developer', got %v", res.Row["name"])
		}

		// BLOB normalized to string
		if rawBytes, ok := res.Row["raw_bytes"].(string); !ok || rawBytes != "byte_data" {
			t.Errorf("expected raw_bytes 'byte_data', got %T (%v)", res.Row["raw_bytes"], res.Row["raw_bytes"])
		}
	})

	t.Run("Executes CTE WITH query", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_cte")

		query := `
			WITH active_users AS (
				SELECT id, name, email FROM users WHERE is_active = @active
			)
			SELECT id, name, email FROM active_users ORDER BY id ASC
		`
		res, err := xsql.Execute(t.Context(), pool, query, map[string]any{"active": 1})
		if err != nil {
			t.Fatalf("unexpected CTE execute error: %v", err)
		}

		if res.RowsAffected != 1 || len(res.Rows) != 1 {
			t.Fatalf("expected 1 active user, got %d", res.RowsAffected)
		}
		if res.Row["name"] != "Jane Developer" {
			t.Errorf("expected user 'Jane Developer', got %v", res.Row["name"])
		}
	})

	t.Run("Executes PRAGMA schema query and returns columns", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_pragma")

		res, err := xsql.Execute(t.Context(), pool, "PRAGMA table_info(users)", nil)
		if err != nil {
			t.Fatalf("unexpected PRAGMA error: %v", err)
		}

		if len(res.Rows) < 5 {
			t.Fatalf("expected at least 5 column definitions from PRAGMA, got %d", len(res.Rows))
		}
	})

	t.Run("Executes INSERT mutation with RETURNING clause", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_returning")

		query := `
			INSERT INTO users (name, email, raw_bytes, created_at, is_active)
			VALUES (@name, @email, X'6e6577', '2026-08-30T14:00:00Z', 1)
			RETURNING id, name, email
		`
		args := map[string]any{"name": "Alice Smith", "email": "alice@example.com"}
		res, err := xsql.Execute(t.Context(), pool, query, args)
		if err != nil {
			t.Fatalf("unexpected RETURNING execute error: %v", err)
		}

		if res.RowsAffected != 1 || res.Row == nil {
			t.Fatalf("expected 1 inserted row returned, got: %+v", res)
		}
		if res.Row["name"] != "Alice Smith" || res.Row["email"] != "alice@example.com" {
			t.Errorf("unexpected returned record: %+v", res.Row)
		}
	})

	t.Run("Executes non-returning UPDATE mutation and tracks RowsAffected", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_update")

		res, err := xsql.Execute(
			t.Context(),
			pool,
			"UPDATE users SET is_active = 1 WHERE is_active = @status",
			map[string]any{"status": 0},
		)
		if err != nil {
			t.Fatalf("unexpected UPDATE execute error: %v", err)
		}

		if res.RowsAffected != 1 {
			t.Errorf("expected 1 row updated, got %d", res.RowsAffected)
		}
		if len(res.Rows) != 0 || res.Row != nil {
			t.Errorf("expected empty rows and nil row for non-returning mutation, got: %+v", res)
		}
	})

	t.Run("Returns empty list and nil row when 0 rows match", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_zero")

		res, err := xsql.Execute(t.Context(), pool, "SELECT * FROM users WHERE id = @id", map[string]any{"id": 99999})
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		if res.RowsAffected != 0 || len(res.Rows) != 0 || res.Row != nil {
			t.Errorf("expected 0 rows and nil row, got: %+v", res)
		}
	})

	t.Run("Fails with error on syntax error in query", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_bad_syntax")

		_, err := xsql.Execute(t.Context(), pool, "INVALID SQL SYNTAX HERE", nil)
		if err == nil {
			t.Fatal("expected error on invalid SQL syntax, got nil")
		}
	})

	t.Run("Aborts execution on canceled context", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_cancel")

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel before execution

		_, err := xsql.Execute(ctx, pool, "SELECT * FROM users", nil)
		if err == nil {
			t.Fatal("expected error on canceled context, got nil")
		}
	})
}
