package sqldb_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/ju4n97/hclapi/internal/sqldb"
)

func setupIsolatedSQLitePool(t *testing.T, dbName string) *sqldb.Pool {
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

	return &sqldb.Pool{
		DB:      db,
		Dialect: sqldb.ResolveDialect("sqlite"),
		Config: sqldb.Config{
			Driver: "sqlite",
			Name:   dbName,
			Source: dsn,
		},
	}
}

func TestPool_Execute(t *testing.T) {
	t.Parallel()

	t.Run("Executes SELECT query with row scanning and type normalization", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_select")

		res, err := pool.Execute(
			t.Context(),
			"SELECT id, name, raw_bytes, created_at FROM users WHERE id = @id",
			map[string]any{"id": 1},
		)
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		if res.RowsAffected != 1 || len(res.Rows) != 1 {
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

	t.Run("Executes CTE query", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_cte")

		query := `
			WITH active_users AS (
				SELECT id, name, email FROM users WHERE is_active = @active
			)
			SELECT id, name, email FROM active_users ORDER BY id ASC
		`
		res, err := pool.Execute(t.Context(), query, map[string]any{"active": 1})
		if err != nil {
			t.Fatalf("unexpected CTE error: %v", err)
		}

		if res.RowsAffected != 1 || len(res.Rows) != 1 {
			t.Fatalf("expected 1 active user, got %d", res.RowsAffected)
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
		res, err := pool.Execute(t.Context(), query, args)
		if err != nil {
			t.Fatalf("unexpected RETURNING error: %v", err)
		}

		if res.RowsAffected != 1 || res.Row == nil {
			t.Fatalf("expected 1 row returned, got: %+v", res)
		}
		if res.Row["name"] != "Alice Smith" || res.Row["email"] != "alice@example.com" {
			t.Errorf("unexpected record: %+v", res.Row)
		}
	})

	t.Run("Executes non-returning UPDATE mutation and tracks RowsAffected", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_update")

		res, err := pool.Execute(
			t.Context(),
			"UPDATE users SET is_active = 1 WHERE is_active = @status",
			map[string]any{"status": 0},
		)
		if err != nil {
			t.Fatalf("unexpected UPDATE error: %v", err)
		}

		if res.RowsAffected != 1 {
			t.Errorf("expected 1 row updated, got %d", res.RowsAffected)
		}
		if len(res.Rows) != 0 || res.Row != nil {
			t.Errorf("expected empty rows for non-returning mutation")
		}
	})

	t.Run("Catches constraint violations via MatchErrorCode", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_constraint")

		// Insert baseline record
		_, err := pool.Execute(
			t.Context(),
			"INSERT INTO users (id, name, email, created_at) VALUES (99, 'Original', 'conflict@example.com', '2026-08-30')",
			nil,
		)
		if err != nil {
			t.Fatalf("failed to insert baseline record: %v", err)
		}

		// Insert duplicate record (conflicting id and unique email)
		_, err = pool.Execute(
			t.Context(),
			"INSERT INTO users (id, name, email, created_at) VALUES (99, 'Duplicate', 'conflict@example.com', '2026-08-30')",
			nil,
		)
		if err == nil {
			t.Fatal("expected duplicate constraint error, got nil")
		}

		code := pool.Dialect.ExtractErrorCode(err)
		if code == "" {
			t.Fatalf("expected non-empty error code from SQLite, got error: %v", err)
		}

		// SQLite unique constraint code is 19 (or 2067 extended)
		if !pool.Dialect.MatchErrorCode(code, "19") && !pool.Dialect.MatchErrorCode(code, "2067") {
			t.Errorf("expected MatchErrorCode to succeed for 19/2067, got code %q (err: %v)", code, err)
		}
	})

	t.Run("Aborts execution on canceled context", func(t *testing.T) {
		t.Parallel()
		pool := setupIsolatedSQLitePool(t, "mem_cancel")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := pool.Execute(ctx, "SELECT * FROM users", nil)
		if err == nil {
			t.Fatal("expected error on canceled context, got nil")
		}
	})
}
