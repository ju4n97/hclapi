// Package engine provides route binding, HTTP multiplexing, and pipeline initialization.
package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
)

// pathParamRegex matches both standard parameters ({id}) and Go 1.22+ catch-all wildcards ({filepath...}).
var pathParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)(?:\.\.\.)?\}`)

// Engine is the root coordinator managing manifests, step registries, and HTTP routing.
type Engine struct {
	options      core.Options
	server       core.Server
	mux          *http.ServeMux
	goSteps      map[string]core.StepHandler
	errorHandler core.ErrorHandler
	logger       *slog.Logger
}

// New initializes an Engine by parsing manifests and registering route endpoints.
func New(options core.Options) (*Engine, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	errorHandler := options.ErrorHandler
	if errorHandler == nil {
		errorHandler = core.DefaultErrorHandler
	}

	manifest, err := parser.Parse(options.ConfigPath, eval.BaseContext())
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifests: %w", err)
	}

	e := &Engine{
		options:      options,
		server:       manifest.Server.ToServer(),
		mux:          http.NewServeMux(),
		goSteps:      make(map[string]core.StepHandler),
		errorHandler: errorHandler,
		logger:       logger,
	}

	for _, ep := range manifest.Endpoints {
		steps, err := parser.DecodePipelineSteps(&ep.Pipeline)
		if err != nil {
			return nil, fmt.Errorf("failed to decode pipeline steps: %w", err)
		}

		e.bindRoute(ep.MethodAndPath, steps)
	}

	return e, nil
}

func (e *Engine) bindRoute(routePattern string, steps []parser.ParsedStep) {
	var paramNames []string
	matches := pathParamRegex.FindAllStringSubmatch(routePattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramNames = append(paramNames, match[1])
		}
	}

	executor := NewPipelineExecutor(steps, e.goSteps)

	e.mux.HandleFunc(routePattern, func(w http.ResponseWriter, r *http.Request) {
		hclapiCtx, err := core.NewContext(w, r,
			core.WithPathParams(paramNames),
			core.WithMaxBodySize(e.server.MaxBodySize.Bytes()),
		)
		if err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				e.logger.WarnContext(r.Context(), "request payload too large", "error", err, "path", r.URL.Path)
				e.errorHandler(w, r, core.ProblemDetailsError{
					Type:   "https://github.com/ju4n97/hclapi/errors/payload-too-large",
					Title:  "Request Entity Too Large",
					Status: http.StatusRequestEntityTooLarge,
					Detail: fmt.Sprintf(
						"request body exceeded maximum size limit of %s",
						e.server.MaxBodySize.String(),
					),
					Instance: r.URL.Path,
				})
				return
			}

			e.logger.WarnContext(r.Context(), "invalid request payload", "error", err, "path", r.URL.Path)
			e.errorHandler(w, r, core.ProblemDetailsError{
				Type:     "https://github.com/ju4n97/hclapi/errors/bad-request",
				Title:    "Invalid Request Payload",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
			return
		}

		if err := executor.Execute(w, hclapiCtx); err != nil {
			e.logger.ErrorContext(r.Context(), "pipeline execution failed", "error", err, "path", r.URL.Path)
			e.errorHandler(w, r, core.ProblemDetailsError{
				Type:     "https://github.com/ju4n97/hclapi/errors/pipeline-execution-failed",
				Title:    "Pipeline Execution Error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
		}
	})
}

// RegisterStep registers a named custom Go function for the pipeline runtime.
func (e *Engine) RegisterStep(name string, handler core.StepHandler) error {
	if _, exists := e.goSteps[name]; exists {
		return fmt.Errorf("step %q is already registered", name)
	}

	e.goSteps[name] = handler
	return nil
}

// Handler returns the underlying http.Handler multiplexer.
func (e *Engine) Handler() http.Handler {
	return e.mux
}

// Server returns the server configuration with defaults applied.
func (e *Engine) Server() core.Server {
	return e.server
}
