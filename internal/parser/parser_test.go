package parser_test

import (
	"path/filepath"
	"testing"

	"github.com/ju4n97/hclapi/internal/parser"
)

func TestParseDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		fixtureDir     string
		expectError    bool
		expectedRoutes int
	}{
		{
			name:           "Valid single file",
			fixtureDir:     "01_valid_basic",
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name:           "Valid multiple files merged into one AST",
			fixtureDir:     "02_multiple_files",
			expectError:    false,
			expectedRoutes: 2,
		},
		{
			name:        "Invalid HCL syntax / type mismatch",
			fixtureDir:  "03_invalid_syntax",
			expectError: true,
		},
		{
			name:        "Missing required block (pipeline)",
			fixtureDir:  "04_missing_block",
			expectError: true,
		},
		{
			name:           "Empty directory returns empty manifest without error",
			fixtureDir:     "05_empty_dir",
			expectError:    false,
			expectedRoutes: 0,
		},
		{
			name:        "Non-existent directory returns error",
			fixtureDir:  "does_not_exist",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dirPath := filepath.Join("testdata", tt.fixtureDir)
			manifest, err := parser.ParseDir(dirPath)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected an error but got nil")
				}
				return // Test passes; error was correctly returned
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
