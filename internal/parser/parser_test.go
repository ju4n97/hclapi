package parser_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/parser"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		name           string
		hcl            string
		expectError    bool
		expectedRoutes int
	}{
		{
			name: "valid basic endpoint",
			hcl: `
				endpoint "GET /health" {
					pipeline {
						respond {
							status = 200
							body   = "OK"
						}
					}
				}
			`,
			expectError:    false,
			expectedRoutes: 1,
		},
		{
			name: "missing required block",
			hcl: `
				endpoint "GET /health" {
					# missing pipeline block
				}
			`,
			expectError: true,
		},
		{
			name: "syntax error",
			hcl: `
				endpoint "GET /health" {
					pipeline {
						respond {
							status = "two hundred" # Should be int
						}
					}
				}
			`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := parser.ParseBytes([]byte(tt.hcl), "test.hcl")

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected an error, but got nil")
				}
				return // Test passed, error was expected
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(manifest.Endpoints) != tt.expectedRoutes {
				t.Errorf("expected %d routes, got %d", tt.expectedRoutes, len(manifest.Endpoints))
			}
		})
	}
}
