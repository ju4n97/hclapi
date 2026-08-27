package core

import "log/slog"

// Options configures runtime parameters for the Hclapi engine.
type Options struct {
	// ManifestDir specifies the filesystem path containing .hcl or Hclapifile manifests.
	ManifestDir string

	// StrictTyping enforces schema validation across all request endpoints.
	StrictTyping bool

	// ErrorHandler allows consumers to override how error responses are formatted.
	// If nil, Hclapi uses the standard RFC 9457 DefaultErrorHandler.
	ErrorHandler ErrorHandler

	// Logger receives structured operational telemetry. If nil, output is discarded.
	Logger *slog.Logger
}
