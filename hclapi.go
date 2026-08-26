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

// Engine is the root coordinator managing manifests, step registries, and HTTP routing.
type Engine struct {
	inner *engine.Engine
}

// NewEngine initializes and boots the Hclapi engine from directory manifests.
func NewEngine(options Options) (*Engine, error) {
	eng, err := engine.New(options)
	if err != nil {
		return nil, err
	}

	return &Engine{inner: eng}, nil
}

// RegisterStep binds a custom Go function for use in pipeline step definitions.
func (e *Engine) RegisterStep(name string, handler func(*Context) (any, error)) error {
	return e.inner.RegisterStep(name, core.StepHandler(handler))
}

// Handler returns the underlying http.Handler to mount Hclapi into an HTTP multiplexer.
func (e *Engine) Handler() http.Handler {
	return e.inner.Handler()
}
