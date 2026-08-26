package core

import "log/slog"

// Options configures runtime parameters for the Hclapi engine.
type Options struct {
	// ManifestDir specifies the filesystem path containing .hcl or Hclapifile manifests.
	ManifestDir string

	// StrictTyping enforces schema validation across all request endpoints.
	StrictTyping bool

	// Logger receives structured operational telemetry. If nil, output is discarded.
	Logger *slog.Logger
}
