package sqldb_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/sqldb"
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
			name:         "Postgres positional placeholders ($1, $2)",
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
			name:         "SQL Server named placeholders (@p1, @p2)",
			driver:       "sqlserver",
			query:        "SELECT name FROM products WHERE sku = @sku AND active = @active",
			args:         map[string]any{"sku": "SKU-99", "active": true},
			expectedSQL:  "SELECT name FROM products WHERE sku = @p1 AND active = @p2",
			expectedArgs: []any{"SKU-99", true},
		},
		{
			name:         "Oracle colon placeholders (:1, :2)",
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
			name:         "Missing argument in map binds nil",
			driver:       "sqlite",
			query:        "SELECT * FROM items WHERE status = @missing_arg",
			args:         map[string]any{},
			expectedSQL:  "SELECT * FROM items WHERE status = ?",
			expectedArgs: []any{nil},
		},
		{
			name:         "Nil args map binds nil for all placeholders",
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

			d := sqldb.ResolveDialect(tt.driver)
			rewritten, args, err := sqldb.RewriteNamedQuery(tt.query, tt.args, d)
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
