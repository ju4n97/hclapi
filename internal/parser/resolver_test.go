package parser_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/parser"
)

func parseHCL(t *testing.T, exprStr string) hcl.Expression {
	t.Helper()

	expr, diags := hclsyntax.ParseExpression(
		[]byte(exprStr),
		"test.hcl",
		hcl.InitialPos,
	)
	if diags.HasErrors() {
		t.Fatalf("failed to parse test expression %q: %s", exprStr, diags.Error())
	}

	return expr
}

func TestResolveConnectionRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		exprStr     string
		expectedRef string
		expectError bool
	}{
		{
			name:        "unquoted standard traversal",
			exprStr:     "connection.postgres.main",
			expectedRef: "connection.postgres.main",
		},
		{
			name:        "unquoted short traversal",
			exprStr:     "postgres.primary",
			expectedRef: "postgres.primary",
		},
		{
			name:        "unquoted deep traversal",
			exprStr:     "connection.db.replica.asia_east",
			expectedRef: "connection.db.replica.asia_east",
		},
		{
			name:        "quoted string literal",
			exprStr:     `"connection.sqlite.main"`,
			expectedRef: "connection.sqlite.main",
		},
		{
			name:        "quoted short string literal",
			exprStr:     `"sqlite.main"`,
			expectedRef: "sqlite.main",
		},
		{
			name:        "empty string literal",
			exprStr:     `""`,
			expectedRef: "",
		},
		{
			name:        "rejects numeric expression",
			exprStr:     "12345",
			expectError: true,
		},
		{
			name:        "rejects boolean expression",
			exprStr:     "true",
			expectError: true,
		},
		{
			name:        "rejects object expression",
			exprStr:     `{ a = "b" }`,
			expectError: true,
		},
		{
			name:        "rejects list expression",
			exprStr:     `["a", "b"]`,
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

	t.Run("rejects nil expression", func(t *testing.T) {
		t.Parallel()

		_, err := parser.ResolveConnectionRef(nil)
		if err == nil {
			t.Fatal("expected error on nil expression, got nil")
		}
	})
}

func TestResolveSchemaRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		exprStr     string
		expectedRef string
		expectError bool
	}{
		{
			name:        "schema traversal",
			exprStr:     "schema.user_create",
			expectedRef: "user_create",
		},
		{
			name:        "schema traversal with underscore",
			exprStr:     "schema.user_create_v2",
			expectedRef: "user_create_v2",
		},
		{
			name:        "schema traversal with multiple attributes",
			exprStr:     "foo.schema.user_create",
			expectedRef: "schema",
		},
		{
			name:        "root-only traversal",
			exprStr:     "user_create",
			expectedRef: "user_create",
		},
		{
			name:        "quoted schema reference",
			exprStr:     `"schema.user_create"`,
			expectedRef: "user_create",
		},
		{
			name:        "quoted schema reference without prefix",
			exprStr:     `"user_create"`,
			expectedRef: "user_create",
		},
		{
			name:        "empty string literal",
			exprStr:     `""`,
			expectedRef: "",
		},
		{
			name:        "non-schema string literal",
			exprStr:     `"connection.postgres"`,
			expectedRef: "connection.postgres",
		},
		{
			name:        "rejects numeric expression",
			exprStr:     "12345",
			expectError: true,
		},
		{
			name:        "rejects boolean expression",
			exprStr:     "true",
			expectError: true,
		},
		{
			name:        "rejects object expression",
			exprStr:     `{ name = "user_create" }`,
			expectError: true,
		},
		{
			name:        "rejects list expression",
			exprStr:     `["user_create"]`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expr := parseHCL(t, tt.exprStr)

			ref, err := parser.ResolveSchemaRef(expr)
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

	t.Run("rejects nil expression", func(t *testing.T) {
		t.Parallel()

		_, err := parser.ResolveSchemaRef(nil)
		if err == nil {
			t.Fatal("expected error on nil expression, got nil")
		}
	})
}
