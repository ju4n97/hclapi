// Package hclapi provides a declarative, embeddable API runtime engine.
package hclapi

import (
	"net/http"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/engine"
)

// Context represents the state passed sequentially across a pipeline execution.
type Context = core.Context

// RequestState holds structured data extracted from the HTTP request.
type RequestState = core.RequestState

// StepResult stores the output or metadata produced by an executed step.
type StepResult = core.StepResult

// StepHandler is the type of a function that executes a single step.
type StepHandler = core.StepHandler

// Options defines the configuration options for the Hclapi engine.
type Options = core.Options

// Duration wraps a time.Duration with universal text deserialization.
type Duration = core.Duration

// ByteSize represents a quantity of bytes that can be unmarshaled from text.
type ByteSize = core.ByteSize

// Server defines the resolved HTTP server configuration.
type Server = core.Server

// Engine is the root coordinator managing manifests, step registries, and HTTP routing.
type Engine struct {
	inner *engine.Engine
}

// NewEngine initializes an Engine by parsing manifests and registering route endpoints.
func NewEngine(options Options) (*Engine, error) {
	eng, err := engine.New(options)
	if err != nil {
		return nil, err
	}

	return &Engine{inner: eng}, nil
}

// RegisterStep registers a named custom Go function for the pipeline runtime.
func (e *Engine) RegisterStep(name string, handler func(*Context) (any, error)) error {
	return e.inner.RegisterStep(name, core.StepHandler(handler))
}

// Handler returns the underlying http.Handler multiplexer.
func (e *Engine) Handler() http.Handler {
	return e.inner.Handler()
}

// Server returns the server configuration with defaults applied.
func (e *Engine) Server() Server {
	return e.inner.Server()
}
