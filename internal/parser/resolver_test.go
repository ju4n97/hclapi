package parser_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/parser"
)

func TestResolveConnectionRef(t *testing.T) {
	t.Parallel()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expr := parseHCL(t, tt.exprStr)
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

func TestResolveSchemaRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		exprStr     string
		expectedRef string
		expectError bool
	}{
		{
			name:        "Unquoted standard traversal (schema.user_create)",
			exprStr:     "schema.user_create",
			expectedRef: "user_create",
			expectError: false,
		},
		{
			name:        "Unquoted root identifier (user_create)",
			exprStr:     "user_create",
			expectedRef: "user_create",
			expectError: false,
		},
		{
			name:        "Quoted string literal with schema prefix (\"schema.user_create\")",
			exprStr:     `"schema.user_create"`,
			expectedRef: "user_create",
			expectError: false,
		},
		{
			name:        "Quoted plain string literal (\"user_create\")",
			exprStr:     `"user_create"`,
			expectedRef: "user_create",
			expectError: false,
		},
		{
			name:        "Rejects numeric expression",
			exprStr:     "999",
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

	t.Run("Rejects nil expression", func(t *testing.T) {
		t.Parallel()

		_, err := parser.ResolveSchemaRef(nil)
		if err == nil {
			t.Fatal("expected error on nil expression, got nil")
		}
	})
}
