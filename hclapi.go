package hclapi

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/ju4n97/hclapi/internal/parser"
	"github.com/ju4n97/hclapi/internal/steps/xgo"
	"github.com/ju4n97/hclapi/internal/steps/xrespond"
	"github.com/ju4n97/hclapi/internal/steps/xstarlark"
)

// pathParamRegex matches path parameter placeholders in a route pattern.
var pathParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

// StepHandler is the type of a function that executes a single step.
type StepHandler func(ctx *Context) (any, error)

// Engine is the core Hclapi runtime. It holds the parsed AST, connection
// pools, and the HTTP multiplexer.
type Engine struct {
	options Options
	mux     *http.ServeMux
	goSteps map[string]StepHandler
	logger  *slog.Logger
}

// NewEngine initializes the Hclapi engine, parses the HCL manifest, and
// builds the routing table.
func NewEngine(options Options) (*Engine, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	logger.Debug("initializing Hclapi engine", "manifest_dir", options.ManifestDir)

	engine := &Engine{
		options: options,
		mux:     http.NewServeMux(),
		goSteps: make(map[string]StepHandler),
		logger:  logger,
	}

	manifest, err := parser.Parse(options.ManifestDir)
	if err != nil {
		logger.Error("failed to parse manifests", "error", err)
		return nil, fmt.Errorf("failed to parse manifests: %w", err)
	}

	logger.Info("manifests loaded", "endpoints_count", len(manifest.Endpoints))

	for _, ep := range manifest.Endpoints {
		steps, err := parser.DecodePipelineSteps(&ep.Pipeline)
		if err != nil {
			logger.Error("failed to decode pipeline steps", "error", err)
			return nil, fmt.Errorf("failed to decode pipeline steps: %w", err)
		}

		engine.bindRoute(ep.MethodAndPath, steps)
	}

	return engine, nil
}

// bindRoute binds a route to the HTTP handler.
func (e *Engine) bindRoute(routePattern string, stepsList []parser.ParsedStep) {
	var paramNames []string
	matches := pathParamRegex.FindAllStringSubmatch(routePattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramNames = append(paramNames, match[1])
		}
	}

	e.logger.Debug("registering route", "pattern", routePattern)

	e.mux.HandleFunc(routePattern, func(w http.ResponseWriter, r *http.Request) {
		e.logger.Debug("request received", "method", r.Method, "path", r.URL.Path)

		ctx := newContext(r, paramNames)
		var lastResult any

		for _, step := range stepsList {
			switch step.Type {
			case parser.StepTypeGo:
				res, err := e.execGoStep(step, ctx)
				if err != nil {
					e.logger.Error("go step failed", "step", step.Name, "error", err)
					http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
					return
				}
				lastResult = res

			case parser.StepTypeStarlark:
				res, err := e.execStarlarkStep(step, ctx)
				if err != nil {
					e.logger.Error("starlark step failed", "step", step.Name, "error", err)
					http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
					return
				}
				lastResult = res

			case parser.StepTypeRespond:
				if err := xrespond.Write(w, step.Respond, lastResult); err != nil {
					e.logger.Error("failed to write response", "error", err)
				}
				return // Terminal step
			}
		}
	})
}

func (e *Engine) execGoStep(step parser.ParsedStep, ctx *Context) (any, error) {
	handler, exists := e.goSteps[step.Go.Use]
	if !exists {
		return nil, fmt.Errorf("unregistered go function %q", step.Go.Use)
	}

	result, err := xgo.Execute(handler, ctx)
	if err != nil {
		return nil, err
	}

	if step.Name != "" {
		ctx.Steps[step.Name] = StepResult{Result: result}
	}

	return result, nil
}

func (e *Engine) execStarlarkStep(step parser.ParsedStep, ctx *Context) (any, error) {
	ctxData := map[string]any{
		"request": map[string]any{
			"method":  ctx.Request.Method,
			"path":    ctx.Request.Path,
			"query":   ctx.Request.Query,
			"headers": ctx.Request.Headers,
			"body":    ctx.Request.Body,
		},
		"steps": make(map[string]any),
	}
	for k, v := range ctx.Steps {
		ctxData["steps"].(map[string]any)[k] = v.Result
	}

	result, err := xstarlark.Eval(step.Starlark.Source, ctxData)
	if err != nil {
		return nil, err
	}

	if step.Name != "" {
		ctx.Steps[step.Name] = StepResult{Result: result}
	}

	return result, nil
}

// RegisterStep registers a native Go function that can be invoked via
// the `go` step block inside an HCL pipeline.
func (e *Engine) RegisterStep(name string, handler func(*Context) (any, error)) error {
	if _, exists := e.goSteps[name]; exists {
		return fmt.Errorf("step '%s' is already registered", name)
	}

	e.goSteps[name] = handler
	return nil
}

// Handler returns the underlying http.Handler so Hclapi can be embedded
// into standard Go web servers.
func (e *Engine) Handler() http.Handler {
	return e.mux
}
