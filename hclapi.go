package hclapi

import (
	"fmt"
	"net/http"

	"github.com/ju4n97/hclapi/internal/parser"
)

// Engine is the core Hclapi runtime. It holds the parsed AST, connection
// pools, and the HTTP multiplexer.
type Engine struct {
	options Options
	mux     *http.ServeMux
	goSteps map[string]func(*Context) (any, error)
}

// NewEngine initializes the Hclapi engine, parses the HCL manifest, and
// builds the routing table.
func NewEngine(options Options) (*Engine, error) {
	engine := &Engine{
		options: options,
		mux:     http.NewServeMux(),
		goSteps: make(map[string]func(*Context) (any, error)),
	}

	manifest, err := parser.ParseDir(options.ManifestDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HCL manifests: %w", err)
	}

	for _, e := range manifest.Endpoints {
		status := e.Pipeline.Respond.Status
		var body string
		if e.Pipeline.Respond.Body != nil {
			body = *e.Pipeline.Respond.Body
		}

		engine.mux.HandleFunc(e.MethodAndPath, func(w http.ResponseWriter, r *http.Request) {
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
