package hclapi

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/ju4n97/hclapi/internal/parser"
)

// Engine is the core Hclapi runtime. It holds the parsed AST, connection
// pools, and the HTTP multiplexer.
type Engine struct {
	options Options
	logger  *slog.Logger
	mux     *http.ServeMux
	goSteps map[string]func(*Context) (any, error)
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
		logger:  logger,
		mux:     http.NewServeMux(),
		goSteps: make(map[string]func(*Context) (any, error)),
	}

	manifest, err := parser.ParseDir(options.ManifestDir)
	if err != nil {
		logger.Error("failed to parse manifests", "error", err)
		return nil, fmt.Errorf("failed to parse manifests: %w", err)
	}

	logger.Info("manifests parsed successfully", "endpoints_found", len(manifest.Endpoints))

	for _, ep := range manifest.Endpoints {
		status := ep.Pipeline.Respond.Status
		var body string
		if ep.Pipeline.Respond.Body != nil {
			body = *ep.Pipeline.Respond.Body
		}

		logger.Debug("registering route", "route", ep.MethodAndPath)

		engine.mux.HandleFunc(ep.MethodAndPath, func(w http.ResponseWriter, r *http.Request) {
			engine.logger.Debug("handling request", "method", r.Method, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			if body != "" {
				w.Write([]byte(body))
			}
		})
	}

	return engine, nil
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
