//go:build integration

package connsql_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/core"
)

func TestIntegration_Postgres(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = pgContainer.Terminate(context.Background())
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	mgr := connsql.NewManager()
	conn := core.Connection{
		Driver: "postgres",
		Name:   "primary",
		URL:    dsn,
		Pool: core.PoolConfig{
			MaxOpenConns:    10,
			MaxIdleConns:    2,
			ConnMaxLifetime: core.Duration(15 * time.Minute),
			IdleTimeout:     core.Duration(5 * time.Minute),
		},
	}
	if err := mgr.Open(ctx, conn); err != nil {
		t.Fatalf("failed to open postgres pool: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	pool, exists := mgr.Get("postgres.primary")
	if !exists {
		t.Fatalf("expected pool to exist for postgres.primary")
	}

	t.Run("Executes DDL and verifies $1, $2 query parameter binding", func(t *testing.T) {
		schema := `
			CREATE TABLE members (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				name VARCHAR(255) NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);
			INSERT INTO members (email, name) VALUES ('jane@example.com', 'Jane');
		`
		if _, err := pool.DB.ExecContext(ctx, schema); err != nil {
			t.Fatalf("failed to seed postgres schema: %v", err)
		}

		var name string
		row := pool.DB.QueryRowContext(ctx, "SELECT name FROM members WHERE email = $1", "jane@example.com")
		if err := row.Scan(&name); err != nil {
			t.Fatalf("failed to scan member: %v", err)
		}
		if name != "Jane" {
			t.Errorf("expected member name 'Jane', got %q", name)
		}
	})

	t.Run("Extracts real unique constraint error code 23505 on duplicate insert", func(t *testing.T) {
		_, err := pool.DB.ExecContext(ctx, "INSERT INTO members (email, name) VALUES ($1, $2)", "jane@example.com", "John")
		if err == nil {
			t.Fatal("expected unique constraint error, got nil")
		}

		code := pool.Dialect.ExtractErrorCode(err)
		if code != "23505" {
			t.Errorf("expected PostgreSQL error code '23505', got %q (raw error: %v)", code, err)
		}

		if !pool.Dialect.MatchErrorCode(code, "23505") {
			t.Errorf("expected MatchErrorCode to return true for 23505")
		}
		if !pool.Dialect.MatchErrorCode(code, "23") {
			t.Errorf("expected MatchErrorCode to return true for class prefix 23")
		}
	})
}

func TestIntegration_MySQL(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("failed to start mysql container: %v", err)
	}
	t.Cleanup(func() {
		_ = mysqlContainer.Terminate(context.Background())
	})

	dsn, err := mysqlContainer.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		t.Fatalf("failed to get mysql connection string: %v", err)
	}

	mgr := connsql.NewManager()
	conn := core.Connection{
		Driver: "mysql",
		Name:   "primary",
		URL:    dsn,
		Pool: core.PoolConfig{
			MaxOpenConns:    10,
			MaxIdleConns:    2,
			ConnMaxLifetime: core.Duration(15 * time.Minute),
			IdleTimeout:     core.Duration(5 * time.Minute),
		},
	}
	if err := mgr.Open(ctx, conn); err != nil {
		t.Fatalf("failed to open mysql pool: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	pool, exists := mgr.Get("mysql.primary")
	if !exists {
		t.Fatalf("expected pool to exist for mysql.primary")
	}

	t.Run("Executes DDL and verifies ? query parameter binding", func(t *testing.T) {
		createStmt := `
			CREATE TABLE customers (
				id INT AUTO_INCREMENT PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				full_name VARCHAR(255) NOT NULL
			);
		`
		if _, err := pool.DB.ExecContext(ctx, createStmt); err != nil {
			t.Fatalf("failed to create mysql table: %v", err)
		}

		insertStmt := "INSERT INTO customers (email, full_name) VALUES ('jane@example.com', 'Jane')"
		if _, err := pool.DB.ExecContext(ctx, insertStmt); err != nil {
			t.Fatalf("failed to insert initial record: %v", err)
		}

		var fullName string
		row := pool.DB.QueryRowContext(ctx, "SELECT full_name FROM customers WHERE email = ?", "jane@example.com")
		if err := row.Scan(&fullName); err != nil {
			t.Fatalf("failed to scan customer: %v", err)
		}
		if fullName != "Jane" {
			t.Errorf("expected customer name 'Jane', got %q", fullName)
		}
	})

	t.Run("Extracts real duplicate entry error code 1062 on duplicate insert", func(t *testing.T) {
		_, err := pool.DB.ExecContext(
			ctx,
			"INSERT INTO customers (email, full_name) VALUES (?, ?)",
			"jane@example.com",
			"John",
		)
		if err == nil {
			t.Fatal("expected duplicate entry error, got nil")
		}

		code := pool.Dialect.ExtractErrorCode(err)
		if code != "1062" {
			t.Errorf("expected MySQL error code '1062', got %q (raw error: %v)", code, err)
		}

		if !pool.Dialect.MatchErrorCode(code, "1062") {
			t.Errorf("expected MatchErrorCode to return true for 1062")
		}
	})
}

func TestIntegration_SQLServer(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mssqlContainer, err := mssql.Run(ctx,
		"mcr.microsoft.com/mssql/server:2022-latest",
		mssql.WithAcceptEULA(),
		mssql.WithPassword("SecretPassword123!"),
	)
	if err != nil {
		t.Fatalf("failed to start mssql container: %v", err)
	}
	t.Cleanup(func() {
		_ = mssqlContainer.Terminate(context.Background())
	})

	dsn, err := mssqlContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get mssql connection string: %v", err)
	}

	mgr := connsql.NewManager()
	conn := core.Connection{
		Driver: "sqlserver",
		Name:   "primary",
		URL:    dsn,
		Pool:   core.DefaultPoolConfig(),
	}
	if err := mgr.Open(ctx, conn); err != nil {
		t.Fatalf("failed to open sqlserver pool: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	pool, exists := mgr.Get("sqlserver.primary")
	if !exists {
		t.Fatalf("expected pool to exist for sqlserver.primary")
	}

	t.Run("Executes DDL and verifies @p1 query parameter binding", func(t *testing.T) {
		createStmt := `
			CREATE TABLE accounts (
				id INT IDENTITY(1,1) PRIMARY KEY,
				email NVARCHAR(255) UNIQUE NOT NULL,
				name NVARCHAR(255) NOT NULL
			);
			INSERT INTO accounts (email, name) VALUES ('jane@example.com', 'Jane');
		`
		if _, err := pool.DB.ExecContext(ctx, createStmt); err != nil {
			t.Fatalf("failed to seed sqlserver schema: %v", err)
		}

		var name string
		row := pool.DB.QueryRowContext(ctx, "SELECT name FROM accounts WHERE email = @p1", "jane@example.com")
		if err := row.Scan(&name); err != nil {
			t.Fatalf("failed to scan account: %v", err)
		}
		if name != "Jane" {
			t.Errorf("expected account name 'Jane', got %q", name)
		}
	})

	t.Run("Extracts real unique constraint error code 2627 on duplicate insert", func(t *testing.T) {
		_, err := pool.DB.ExecContext(ctx, "INSERT INTO accounts (email, name) VALUES (@p1, @p2)", "jane@example.com", "John")
		if err == nil {
			t.Fatal("expected unique constraint error on sqlserver, got nil")
		}

		code := pool.Dialect.ExtractErrorCode(err)
		if code != "2627" && code != "2601" {
			t.Errorf("expected SQL Server error code '2627' or '2601', got %q (raw error: %v)", code, err)
		}

		if !pool.Dialect.MatchErrorCode(code, "2627") && !pool.Dialect.MatchErrorCode(code, "2601") {
			t.Errorf("expected MatchErrorCode to return true for 2627/2601")
		}
	})
}
