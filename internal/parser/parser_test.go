package parser_test

import (
	"path/filepath"
	"testing"

	"github.com/ju4n97/hclapi/internal/parser"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		targetPath     string
		expectError    bool
		expectedRoutes int
	}{
		{
			name:           "Case 01: Flat single file in directory",
			targetPath:     "01_flat_single_file",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:           "Case 02: Deeply nested directory tree merged into one AST",
			targetPath:     "02_nested_tree",
			expectError:    false,
			expectedRoutes: 4, // 1 in root Hclapifile + 2 in v1/users + 1 in v2/orders
		},
		{
			name:           "Case 03: Mixed extensions (Hclapifile, .hclapi, .hcl) and ignores non-manifests",
			targetPath:     "03_mixed_extensions",
			expectError:    false,
			expectedRoutes: 3,
		},
		{
			name:           "Case 04: Hidden directories (.git) are skipped completely",
			targetPath:     "04_hidden_dir_ignored",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:        "Case 05: Invalid HCL syntax / type mismatch",
			targetPath:  "05_syntax_error",
			expectError: true,
		},
		{
			name:        "Case 06: Missing required pipeline block",
			targetPath:  "06_missing_block",
			expectError: true,
		},
		{
			name:           "Case 07: Empty directory tree returns empty manifest without error",
			targetPath:     "07_empty_tree",
			expectError:    false,
			expectedRoutes: 0,
		},
		{
			name:           "Direct file target: Single HCL file path",
			targetPath:     "01_flat_single_file/main.hcl",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:           "Direct file target: Extensionless Hclapifile path",
			targetPath:     "02_nested_tree/Hclapifile",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:        "Non-existent path returns error",
			targetPath:  "does_not_exist",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target := filepath.Join("testdata", tt.targetPath)
			manifest, err := parser.Parse(target)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected an error but got nil")
				}
				return // Expected error occurred, test passed
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if manifest == nil {
				t.Fatalf("expected manifest to not be nil")
			}

			if len(manifest.Endpoints) != tt.expectedRoutes {
				t.Errorf("expected %d routes, got %d", tt.expectedRoutes, len(manifest.Endpoints))
			}
		})
	}
}
