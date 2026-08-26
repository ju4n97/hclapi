// Package engine provides route binding, HTTP multiplexing, and pipeline initialization.
package engine

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/parser"
)

var pathParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

// Engine is the central HTTP coordinator.
type Engine struct {
	options core.Options
	mux     *http.ServeMux
	goSteps map[string]core.StepHandler
	logger  *slog.Logger
}

// New initializes an Engine by parsing manifests and registering route endpoints.
func New(options core.Options) (*Engine, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	e := &Engine{
		options: options,
		mux:     http.NewServeMux(),
		goSteps: make(map[string]core.StepHandler),
		logger:  logger,
	}

	manifest, err := parser.Parse(options.ManifestDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifests: %w", err)
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
		ctx := core.NewContext(r, paramNames)
		if err := executor.Execute(w, ctx); err != nil {
			e.logger.Error("pipeline execution failed", "error", err)
			http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
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
