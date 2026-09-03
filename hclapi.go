// Package hclapi provides a declarative, embeddable API runtime engine.
package hclapi

import (
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/engine"
)

// Engine is the root coordinator managing manifests, step registries, and HTTP routing.
type Engine = engine.Engine

// ExecutionContext encapsulates the runtime state for a single HTTP request pipeline execution.
type ExecutionContext = core.ExecutionContext

// ExecutionContextOption configures optional behavior during ExecutionContext creation.
type ExecutionContextOption = core.ExecutionContextOption

// Step encapsulates the invocation state and arguments for a single Go step.
type Step = core.Step

// StepHandler defines the signature for custom native Go step callbacks.
type StepHandler = core.StepHandler

// Args represents evaluated arguments passed to a Go step from an HCL manifest.
type Args = core.Args

// RequestState represents normalized HTTP request metadata extracted at runtime.
type RequestState = core.RequestState

// StepResult represents arbitrary step-specific outputs.
type StepResult = core.StepResult

// Options defines the configuration options for the hclapi engine.
type Options = core.Options

// Server defines the resolved HTTP server configuration.
type Server = core.Server

// Duration wraps a time.Duration with universal text deserialization.
type Duration = core.Duration

// ByteSize represents a quantity of bytes that can be unmarshaled from text.
type ByteSize = core.ByteSize

// ProblemDetailsError represents an RFC 9457 compliant error object.
type ProblemDetailsError = core.ProblemDetailsError

// InvalidParam represents a single field validation failure.
type InvalidParam = core.InvalidParam

// ErrorHandler defines the contract for customizing API error serialization.
type ErrorHandler = core.ErrorHandler

// DefaultErrorHandler returns a ProblemDetails with default values.
var DefaultErrorHandler = core.DefaultErrorHandler

// NewEngine initializes an Engine by parsing manifests and registering route endpoints.
func NewEngine(options Options) (*Engine, error) {
	return engine.New(options)
}
