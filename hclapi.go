package hclapi

import (
	"fmt"
	"net/http"
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
	e := &Engine{
		options: options,
		mux:     http.NewServeMux(),
		goSteps: make(map[string]func(*Context) (any, error)),
	}

	// TODO: parse HCL manifests and create routing table
	e.mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "hclapi_engine_active", "version": "0.1.0-dev"}`))
	})

	return e, nil
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
