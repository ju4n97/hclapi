package parser_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/parser"
)

func TestResolveConnectionRef(t *testing.T) {
	t.Parallel()

	parseHCL := func(t *testing.T, exprStr string) hcl.Expression {
		t.Helper()
		expr, diags := hclsyntax.ParseExpression([]byte(exprStr), "test.hcl", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("failed to parse test expression %q: %s", exprStr, diags.Error())
		}
		return expr
	}

	tests := []struct {
		name        string
		exprStr     string
		expectedRef string
		expectError bool
	}{
		{
			name:        "Unquoted standard traversal (connection.postgres.main)",
			exprStr:     "connection.postgres.main",
			expectedRef: "connection.postgres.main",
			expectError: false,
		},
		{
			name:        "Unquoted short traversal (postgres.primary)",
			exprStr:     "postgres.primary",
			expectedRef: "postgres.primary",
			expectError: false,
		},
		{
			name:        "Unquoted deep traversal (connection.db.replica.asia_east)",
			exprStr:     "connection.db.replica.asia_east",
			expectedRef: "connection.db.replica.asia_east",
			expectError: false,
		},
		{
			name:        "Quoted string literal (\"connection.sqlite.main\")",
			exprStr:     `"connection.sqlite.main"`,
			expectedRef: "connection.sqlite.main",
			expectError: false,
		},
		{
			name:        "Quoted short string literal (\"sqlite.main\")",
			exprStr:     `"sqlite.main"`,
			expectedRef: "sqlite.main",
			expectError: false,
		},
		{
			name:        "Rejects numeric expressions",
			exprStr:     "12345",
			expectError: true,
		},
		{
			name:        "Rejects object map expressions",
			exprStr:     `{ a = "b" }`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var expr hcl.Expression
			if tt.exprStr != "" {
				expr = parseHCL(t, tt.exprStr)
			}

			ref, err := parser.ResolveConnectionRef(expr)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for %q, got nil (ref: %q)", tt.exprStr, ref)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ref != tt.expectedRef {
				t.Errorf("expected ref %q, got %q", tt.expectedRef, ref)
			}
		})
	}

	t.Run("Rejects nil expression", func(t *testing.T) {
		t.Parallel()

		_, err := parser.ResolveConnectionRef(nil)
		if err == nil {
			t.Fatal("expected error on nil expression, got nil")
		}
	})
}
