package core_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
)

func TestResolveRelativePath(t *testing.T) {
	t.Parallel()

	baseDir := "/home/user/project/config"

	tests := []struct {
		name     string
		rawPath  string
		baseDir  string
		expected string
	}{
		{
			name:     "Plain relative file path",
			rawPath:  "data/todos.db",
			baseDir:  baseDir,
			expected: filepath.Join(baseDir, "data/todos.db"),
		},
		{
			name:     "Relative file path with dot-slash",
			rawPath:  "./data/todos.db",
			baseDir:  baseDir,
			expected: filepath.Join(baseDir, "./data/todos.db"),
		},
		{
			name:     "File URI with relative path and query parameters",
			rawPath:  "file:./data/todos.db?mode=rwc&_journal_mode=WAL",
			baseDir:  baseDir,
			expected: "file:" + filepath.Join(baseDir, "./data/todos.db") + "?mode=rwc&_journal_mode=WAL",
		},
		{
			name:     "File URI with simple relative path",
			rawPath:  "file:todos.db",
			baseDir:  baseDir,
			expected: "file:" + filepath.Join(baseDir, "todos.db"),
		},
		{
			name:     "In-memory sqlite colon notation",
			rawPath:  ":memory:",
			baseDir:  baseDir,
			expected: ":memory:",
		},
		{
			name:     "In-memory file URI notation",
			rawPath:  "file::memory:?cache=shared",
			baseDir:  baseDir,
			expected: "file::memory:?cache=shared",
		},
		{
			name:     "Absolute path remains unchanged",
			rawPath:  "/var/lib/data/app.db",
			baseDir:  baseDir,
			expected: "/var/lib/data/app.db",
		},
		{
			name:     "Network DSN (postgres://) remains unchanged",
			rawPath:  "postgres://user:pass@localhost:5432/db",
			baseDir:  baseDir,
			expected: "postgres://user:pass@localhost:5432/db",
		},
		{
			name:     "Network DSN (redis://) remains unchanged",
			rawPath:  "redis://localhost:6379/0",
			baseDir:  baseDir,
			expected: "redis://localhost:6379/0",
		},
		{
			name:     "Empty baseDir returns raw path unchanged",
			rawPath:  "./data/todos.db",
			baseDir:  "",
			expected: "./data/todos.db",
		},
		{
			name:     "Non-file path like WASM module relative path",
			rawPath:  "./plugins/crypto.wasm",
			baseDir:  baseDir,
			expected: filepath.Join(baseDir, "./plugins/crypto.wasm"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := core.ResolveRelativePath(tt.rawPath, tt.baseDir)
			// Normalize path separators for cross-platform test assertions (Windows vs Unix)
			actualNormalized := filepath.Clean(actual)
			expectedNormalized := filepath.Clean(tt.expected)

			// Preserve "file:" prefix and query string structure if present during comparison
			if strings.HasPrefix(tt.expected, "file:") {
				if !strings.HasPrefix(actual, "file:") {
					t.Errorf("expected 'file:' prefix, got %q", actual)
				}
			}

			if actualNormalized != expectedNormalized {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
