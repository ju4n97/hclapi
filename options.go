package hclapi

import "log/slog"

// Options defines the configuration for the Hclapi engine.
type Options struct {
	// ManifestDir is the path to the directory containing .hcl files.
	ManifestDir string

	// StrictTyping ensures all schema references are strictly validated.
	StrictTyping bool

	// Logger allows embedded applications to inject their own structured logger.
	// If nil, Hclapi will use a default discard or standard logger.
	Logger *slog.Logger
}
