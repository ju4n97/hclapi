//go:build integration

package connsql_test

import (
	"context"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/scalar"
	"github.com/ju4n97/hclapi/internal/steps/xsql"
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
	conn := manifest.Connection{
		Driver: "postgres",
		Name:   "primary",
		Source: dsn,
		Pool: manifest.PoolConfig{
			MaxOpen:     10,
			MaxIdle:     2,
			MaxLifetime: scalar.Duration(15 * time.Minute),
			IdleTimeout: scalar.Duration(5 * time.Minute),
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

	t.Run("Executes DDL, parameter queries, and RETURNING clause", func(t *testing.T) {
		createStmt := `
			CREATE TABLE members (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				name VARCHAR(255) NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);
		`
		if _, err := pool.DB.ExecContext(ctx, createStmt); err != nil {
			t.Fatalf("failed to create postgres table: %v", err)
		}

		// Insert with RETURNING
		insertQuery := "INSERT INTO members (email, name) VALUES (@email, @name) RETURNING id, email, name"
		res, err := xsql.Execute(ctx, pool, insertQuery, map[string]any{"email": "jane@example.com", "name": "Jane Developer"})
		if err != nil {
			t.Fatalf("failed to insert member: %v", err)
		}
		if res.RowsAffected != 1 || res.Row["name"] != "Jane Developer" {
			t.Errorf("unexpected inserted record: %+v", res)
		}

		// Query with $1 placeholder
		selectQuery := "SELECT name FROM members WHERE email = @email"
		selectRes, err := xsql.Execute(ctx, pool, selectQuery, map[string]any{"email": "jane@example.com"})
		if err != nil {
			t.Fatalf("failed to select member: %v", err)
		}
		if selectRes.Row["name"] != "Jane Developer" {
			t.Errorf("expected 'Jane Developer', got %v", selectRes.Row["name"])
		}
	})

	t.Run("Executes PostgreSQL stored procedure with CALL", func(t *testing.T) {
		procStmt := `
			CREATE OR REPLACE PROCEDURE update_member_name(p_id INT, p_name VARCHAR)
			LANGUAGE plpgsql AS $$
			BEGIN
				UPDATE members SET name = p_name WHERE id = p_id;
			END;
			$$;
		`
		if _, err := pool.DB.ExecContext(ctx, procStmt); err != nil {
			t.Fatalf("failed to create postgres stored procedure: %v", err)
		}

		_, err := xsql.Execute(ctx, pool, "CALL update_member_name(@id, @name)", map[string]any{"id": 1, "name": "Jane Updated"})
		if err != nil {
			t.Fatalf("failed to execute postgres procedure: %v", err)
		}

		// Verify the database record was updated by the stored procedure
		var updatedName string
		row := pool.DB.QueryRowContext(ctx, "SELECT name FROM members WHERE id = $1", 1)
		if err := row.Scan(&updatedName); err != nil {
			t.Fatalf("failed to verify updated member: %v", err)
		}
		if updatedName != "Jane Updated" {
			t.Errorf("expected name 'Jane Updated' after stored procedure call, got %q", updatedName)
		}
	})

	t.Run("Extracts real unique constraint error code 23505 on duplicate insert", func(t *testing.T) {
		_, err := pool.DB.ExecContext(ctx, "INSERT INTO members (email, name) VALUES ($1, $2)", "jane@example.com", "John Doe")
		if err == nil {
			t.Fatal("expected unique constraint error, got nil")
		}

		code := pool.Dialect.ExtractErrorCode(err)
		if code != "23505" {
			t.Errorf("expected PostgreSQL error code '23505', got %q", code)
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
	conn := manifest.Connection{
		Driver: "mysql",
		Name:   "primary",
		Source: dsn,
		Pool: manifest.PoolConfig{
			MaxOpen:     10,
			MaxIdle:     2,
			MaxLifetime: scalar.Duration(15 * time.Minute),
			IdleTimeout: scalar.Duration(5 * time.Minute),
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

	t.Run("Executes DDL, parameter queries, and stored procedure with result set", func(t *testing.T) {
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

		insertQuery := "INSERT INTO customers (email, full_name) VALUES (@email, @name)"
		res, err := xsql.Execute(ctx, pool, insertQuery, map[string]any{"email": "jane@example.com", "name": "Jane Developer"})
		if err != nil {
			t.Fatalf("failed to insert customer: %v", err)
		}
		if res.RowsAffected != 1 {
			t.Errorf("expected 1 row affected, got %d", res.RowsAffected)
		}

		// Create Stored Procedure (Single statement)
		_, _ = pool.DB.ExecContext(ctx, "DROP PROCEDURE IF EXISTS award_member_points")
		procStmt := `
			CREATE PROCEDURE award_member_points(IN p_id INT, IN p_bonus INT)
			BEGIN
				UPDATE customers SET full_name = CONCAT(full_name, ' (VIP)') WHERE id = p_id;
				SELECT id, full_name, p_bonus AS bonus_awarded FROM customers WHERE id = p_id;
			END
		`
		if _, err := pool.DB.ExecContext(ctx, procStmt); err != nil {
			t.Fatalf("failed to create mysql stored procedure: %v", err)
		}

		// Call stored procedure and scan returned result set
		callRes, err := xsql.Execute(ctx, pool, "CALL award_member_points(@id, @bonus)", map[string]any{"id": 1, "bonus": 500})
		if err != nil {
			t.Fatalf("failed to execute mysql procedure: %v", err)
		}
		if callRes.RowsAffected != 1 || callRes.Row == nil {
			t.Fatalf("expected 1 row returned from procedure, got: %+v", callRes)
		}
		if callRes.Row["full_name"] != "Jane Developer (VIP)" {
			t.Errorf("expected full_name 'Jane Developer (VIP)', got %v", callRes.Row["full_name"])
		}
	})

	t.Run("Extracts real duplicate entry error code 1062 on duplicate insert", func(t *testing.T) {
		_, err := pool.DB.ExecContext(
			ctx,
			"INSERT INTO customers (email, full_name) VALUES (?, ?)",
			"jane@example.com",
			"John Doe",
		)
		if err == nil {
			t.Fatal("expected duplicate entry error, got nil")
		}

		code := pool.Dialect.ExtractErrorCode(err)
		if code != "1062" {
			t.Errorf("expected MySQL error code '1062', got %q", code)
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
	conn := manifest.Connection{
		Driver: "sqlserver",
		Name:   "primary",
		Source: dsn,
		Pool:   manifest.DefaultPoolConfig(),
	}
	if err := mgr.Open(ctx, conn); err != nil {
		t.Fatalf("failed to open sqlserver pool: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	pool, exists := mgr.Get("sqlserver.primary")
	if !exists {
		t.Fatalf("expected pool to exist for sqlserver.primary")
	}

	t.Run("Executes DDL, OUTPUT INSERTED clause, and stored procedure", func(t *testing.T) {
		createStmt := `
			CREATE TABLE accounts (
				id INT IDENTITY(1,1) PRIMARY KEY,
				email NVARCHAR(255) UNIQUE NOT NULL,
				name NVARCHAR(255) NOT NULL
			);
		`
		if _, err := pool.DB.ExecContext(ctx, createStmt); err != nil {
			t.Fatalf("failed to create sqlserver table: %v", err)
		}

		// Insert with OUTPUT INSERTED.*
		insertQuery := `
			INSERT INTO accounts (email, name)
			OUTPUT INSERTED.id, INSERTED.email, INSERTED.name
			VALUES (@email, @name)
		`
		res, err := xsql.Execute(ctx, pool, insertQuery, map[string]any{"email": "jane@example.com", "name": "Jane Developer"})
		if err != nil {
			t.Fatalf("failed to insert record with OUTPUT: %v", err)
		}
		if res.RowsAffected != 1 || res.Row["name"] != "Jane Developer" {
			t.Errorf("unexpected inserted record: %+v", res)
		}

		// Create Stored Procedure
		procStmt := `
			CREATE PROCEDURE get_account_details
				@email NVARCHAR(255)
			AS
			BEGIN
				SET NOCOUNT ON;
				SELECT id, email, name FROM accounts WHERE email = @email;
			END;
		`
		if _, err := pool.DB.ExecContext(ctx, procStmt); err != nil {
			t.Fatalf("failed to create sqlserver procedure: %v", err)
		}

		// Execute Stored Procedure via EXEC
		execRes, err := xsql.Execute(ctx, pool, "EXEC get_account_details @email", map[string]any{"email": "jane@example.com"})
		if err != nil {
			t.Fatalf("failed to execute sqlserver procedure: %v", err)
		}
		if execRes.Row["name"] != "Jane Developer" {
			t.Errorf("expected 'Jane Developer', got %v", execRes.Row["name"])
		}
	})

	t.Run("Extracts real unique constraint error code 2627 on duplicate insert", func(t *testing.T) {
		_, err := pool.DB.ExecContext(ctx, "INSERT INTO accounts (email, name) VALUES (@p1, @p2)", "jane@example.com", "John Doe")
		if err == nil {
			t.Fatal("expected unique constraint error on sqlserver, got nil")
		}

		code := pool.Dialect.ExtractErrorCode(err)
		if code != "2627" && code != "2601" {
			t.Errorf("expected SQL Server error code '2627' or '2601', got %q", code)
		}

		if !pool.Dialect.MatchErrorCode(code, "2627") && !pool.Dialect.MatchErrorCode(code, "2601") {
			t.Errorf("expected MatchErrorCode to return true for 2627/2601")
		}
	})
}
