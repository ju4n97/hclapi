package manifest

import (
	"log/slog"

	"github.com/ju4n97/hclapi/internal/problem"
)

// Options defines the configuration options for the hclapi engine.
type Options struct {
	// ConfigPath is a file or directory of .hcl definitions.
	ConfigPath string

	// StrictTyping enforces request schema validation on all endpoints.
	StrictTyping bool

	// ProblemHandler formats error responses. If nil, RFC 9457 defaults are used.
	ProblemHandler problem.Handler

	// Logger receives operational logs. If nil, logging is discarded.
	Logger *slog.Logger
}
