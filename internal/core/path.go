package core

import (
	"path/filepath"
	"strings"
)

// ResolveRelativePath resolves a relative file path or file-based DSN against baseDir.
func ResolveRelativePath(raw, baseDir string) string {
	if baseDir == "" || filepath.IsAbs(raw) || raw == ":memory:" || strings.Contains(raw, "://") {
		return raw
	}

	// Handle URI-style file connections (e.g., file:./data/todos.db?mode=rwc)
	if strings.HasPrefix(raw, "file:") {
		rest := strings.TrimPrefix(raw, "file:")
		pathPart := rest
		queryPart := ""
		if idx := strings.Index(rest, "?"); idx != -1 {
			pathPart = rest[:idx]
			queryPart = rest[idx:]
		}

		if pathPart == ":memory:" || strings.Contains(pathPart, ":memory:") {
			return raw
		}

		absPath := pathPart
		if !filepath.IsAbs(pathPart) {
			absPath = filepath.Join(baseDir, pathPart)
		}
		return "file:" + filepath.ToSlash(absPath) + queryPart
	}

	// Handle plain relative file paths (e.g., ./data/db.sqlite, plugins/crypto.wasm)
	if !filepath.IsAbs(raw) {
		return filepath.Join(baseDir, raw)
	}

	return raw
}
