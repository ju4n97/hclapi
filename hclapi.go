// Package hclapi provides a declarative, embeddable API runtime engine.
package hclapi

import (
	"github.com/ju4n97/hclapi/internal/engine"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
)

type (
	// Engine coordinates manifest execution, connection pools, and HTTP routing.
	Engine = engine.Engine

	// Options defines configuration parameters for initializing an Engine.
	Options = engine.Options

	// Step provides invocation arguments and request context to a Go step callback.
	Step = runtime.Step

	// StepHandler defines the signature for custom native Go step callbacks.
	StepHandler = runtime.StepHandler

	// Args represents evaluated arguments passed to a Go step callback.
	Args = runtime.Args

	// ExecutionContext encapsulates the state for a single HTTP pipeline run.
	ExecutionContext = runtime.ExecutionContext

	// RequestState holds normalized, read-only HTTP request metadata.
	RequestState = runtime.RequestState

	// Problem represents an RFC 9457 compliant error object.
	Problem = problem.Problem

	// ProblemHandler defines the contract for serializing Problem Details to an HTTP client.
	ProblemHandler = problem.Handler

	// InvalidParam captures a single field validation constraint failure.
	InvalidParam = problem.InvalidParam
)

// DefaultProblemHandler formats and serializes errors as application/problem+json.
var DefaultProblemHandler = problem.DefaultHandler

// New compiles manifests and initializes the HTTP engine.
func New(options Options) (*Engine, error) {
	return engine.New(options)
}

// NewProblem constructs a Problem with canonical title and type derived from the status code.
func NewProblem(status int, detail ...string) Problem {
	return problem.New(status, detail...)
}
