// Package hclapi provides a declarative, embeddable API runtime engine.
package hclapi

import (
	"github.com/ju4n97/hclapi/internal/engine"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
	"github.com/ju4n97/hclapi/internal/scalar"
)

// Engine is the root coordinator managing manifests, step registries, and HTTP routing.
type Engine = engine.Engine

// Runtime

// Step encapsulates the invocation state, arguments, and request metadata for a Go step.
type Step = runtime.Step

// StepHandler defines the signature for custom native Go step callbacks.
type StepHandler = runtime.StepHandler

// Args represents evaluated arguments passed to a Go step from an HCL manifest.
type Args = runtime.Args

// ExecutionContext encapsulates the runtime state for a single HTTP request pipeline execution.
type ExecutionContext = runtime.ExecutionContext

// ExecutionContextOption configures optional behavior during ExecutionContext creation.
type ExecutionContextOption = runtime.ExecutionContextOption

// RequestState represents normalized HTTP request metadata extracted at runtime.
type RequestState = runtime.RequestState

// StepResult represents arbitrary step-specific outputs.
type StepResult = runtime.StepResult

// Manifest models

// Options defines configuration parameters for the hclapi engine.
type Options = manifest.Options

// Server defines the resolved HTTP server transport configuration.
type Server = manifest.Server

// Connection represents a resolved database, cache, or storage backend configuration.
type Connection = manifest.Connection

// PoolConfig defines connection pool sizing and lifecycle settings.
type PoolConfig = manifest.PoolConfig

// Schema represents a compiled, named validation schema.
type Schema = manifest.Schema

// Field represents a compiled, type-safe schema field constraint rule.
type Field = manifest.Field

// Scalar unit types

// Duration wraps a time.Duration with human-readable text deserialization (e.g. "15m", "30s").
type Duration = scalar.Duration

// ByteSize represents a quantity of bytes unmarshaled from text (e.g. "25MB", "10GiB").
type ByteSize = scalar.ByteSize

// Problem

// Problem represents an RFC 9457 compliant error object.
type Problem = problem.Problem

// NewProblem creates a Problem with title and type derived from the HTTP status code.
func NewProblem(status int, detail ...string) Problem {
	return problem.New(status, detail...)
}

// ProblemHandler defines the contract for custom error serialization.
type ProblemHandler = problem.Handler

// DefaultProblemHandler serializes Problem as application/problem+json.
var DefaultProblemHandler = problem.DefaultHandler

// InvalidParam represents a single field-level schema validation constraint failure.
type InvalidParam = problem.InvalidParam

// NewEngine parses manifests, statically verifies routes, and initializes the HTTP engine.
func NewEngine(options Options) (*Engine, error) {
	return engine.New(options)
}
