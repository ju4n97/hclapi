// Package hclapi provides a declarative, embeddable API runtime engine.
package hclapi

import (
	"fmt"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/engine"
)

// Engine is the root coordinator managing manifests, step registries, and HTTP routing.
type Engine = engine.Engine

// Context represents the state passed sequentially across a pipeline execution.
type Context = core.Context

// ContextOption configures optional behavior during Context creation.
type ContextOption = core.ContextOption

// RequestState represents normalized HTTP request metadata extracted at runtime.
type RequestState = core.RequestState

// StepResult represents arbitrary step-specific outputs.
type StepResult = core.StepResult

// StepHandler defines the signature for custom native Go step callbacks.
type StepHandler = core.StepHandler

// Options defines the configuration options for the hclapi engine.
type Options = core.Options

// Duration wraps a time.Duration with universal text deserialization.
type Duration = core.Duration

// ByteSize represents a quantity of bytes that can be unmarshaled from text.
type ByteSize = core.ByteSize

// Server defines the resolved HTTP server configuration.
type Server = core.Server

// ProblemDetails represents an RFC 9457 compliant error object.
type ProblemDetailsError = core.ProblemDetailsError

// InvalidParam represents a single field validation failure.
type InvalidParam = core.InvalidParam

// ErrorHandler defines the contract for customizing API error serialization.
type ErrorHandler = core.ErrorHandler

// DefaultErrorHandler returns a ProblemDetails with default values.
var DefaultErrorHandler = core.DefaultErrorHandler

// NewEngine initializes an Engine by parsing manifests and registering route endpoints.
func NewEngine(options Options) (*Engine, error) {
	eng, err := engine.New(options)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize hclapi engine: %w", err)
	}

	return eng, nil
}
