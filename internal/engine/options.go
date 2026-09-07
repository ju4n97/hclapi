package engine

import (
	"log/slog"

	"github.com/ju4n97/hclapi/internal/problem"
)

// Options defines configuration parameters for initializing the engine.
type Options struct {
	// ConfigPath is a file or directory path containing .hcl manifests.
	ConfigPath string

	// StrictTyping enforces request schema validation on all endpoints.
	StrictTyping bool

	// ProblemHandler customizes RFC 9457 error serialization. If nil, defaults are used.
	ProblemHandler problem.Handler

	// Logger receives operational engine logs. If nil, logs are discarded.
	Logger *slog.Logger
}
