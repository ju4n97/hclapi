package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/config"
	"github.com/ju4n97/hclapi/internal/eval"
)

func writeConfigFile(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", relPath, err)
		}
	}
	return tmpDir
}

func TestLoad_DuplicateSingletons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		files       map[string]string
		expectError string
	}{
		{
			name: "duplicate server block across files fails at boot",
			files: map[string]string{
				"server_a.hcl": `server { port = 8080 }`,
				"server_b.hcl": `server { port = 9000 }`,
			},
			expectError: "duplicate singleton block 'server'",
		},
		{
			name: "duplicate openapi block across files fails at boot",
			files: map[string]string{
				"openapi_a.hcl": `openapi { title = "A" }`,
				"openapi_b.hcl": `openapi { title = "B" }`,
			},
			expectError: "duplicate singleton block 'openapi'",
		},
		{
			name: "duplicate problem block across files fails at boot",
			files: map[string]string{
				"problem_a.hcl": `problem { type_prefix = "https://a.com/" }`,
				"problem_b.hcl": `problem { type_prefix = "https://b.com/" }`,
			},
			expectError: "duplicate singleton block 'problem'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := writeConfigFile(t, tt.files)
			_, err := config.Load(dir, eval.BaseContext())
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectError)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected error to contain %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}
