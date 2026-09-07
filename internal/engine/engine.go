package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/ju4n97/hclapi/internal/config"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/openapi"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
	"github.com/ju4n97/hclapi/internal/sqldb"
)

var pathParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)(?:\.{3})?\}`)

// Engine is the central HTTP coordinator managing routing, pools, and pipelines.
type Engine struct {
	options    Options
	config     *config.Config
	mux        *http.ServeMux
	sqlManager *sqldb.Manager
	registry   *StepRegistry
	errors     *ErrorResponder
	logger     *slog.Logger
}

// New initializes an Engine by loading and verifying manifests from ConfigPath.
func New(options Options) (*Engine, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	evalCtx := eval.BaseContext()
	cfg, err := config.Load(options.ConfigPath, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	bootCtx := context.Background()
	sqlManager := sqldb.NewManager()

	for _, conn := range cfg.Connections {
		if sqldb.IsSupportedDriver(conn.Driver) {
			dbCfg := sqldb.Config{
				Driver: conn.Driver,
				Name:   conn.Name,
				Source: conn.Source,
				Pool: sqldb.PoolConfig{
					MaxOpen:     conn.Pool.MaxOpen,
					MaxIdle:     conn.Pool.MaxIdle,
					MaxLifetime: conn.Pool.MaxLifetime.Duration(),
					IdleTimeout: conn.Pool.IdleTimeout.Duration(),
				},
			}
			if err := sqlManager.Open(bootCtx, dbCfg); err != nil {
				_ = sqlManager.Close()
				return nil, fmt.Errorf("init connection %q: %w", conn.Reference(), err)
			}
			logger.Info("initialized database connection pool", "connection", conn.Reference(), "driver", conn.Driver)
		}
	}

	registry := NewStepRegistry()
	errorResponder := NewErrorResponder(cfg.Problem, options.ProblemHandler, logger)

	e := &Engine{
		options:    options,
		config:     cfg,
		mux:        http.NewServeMux(),
		sqlManager: sqlManager,
		registry:   registry,
		errors:     errorResponder,
		logger:     logger,
	}

	// Precompute OpenAPI specifications and ETags
	var (
		specJSON     []byte
		specYAML     []byte
		specJSONETag string
		specYAMLETag string
	)

	hasOpenAPI := false
	for _, endpoint := range cfg.Endpoints {
		if _, ok := endpoint.Handler.(config.OpenAPIHandler); ok {
			hasOpenAPI = true
			break
		}
	}

	if hasOpenAPI {
		specJSON, err = openapi.GenerateJSON(cfg, true)
		if err != nil {
			return nil, fmt.Errorf("generate OpenAPI 3.1 JSON: %w", err)
		}
		specYAML, err = openapi.GenerateYAML(cfg)
		if err != nil {
			return nil, fmt.Errorf("generate OpenAPI 3.1 YAML: %w", err)
		}

		hJSON := sha256.Sum256(specJSON)
		specJSONETag = fmt.Sprintf("%q", hex.EncodeToString(hJSON[:]))
		hYAML := sha256.Sum256(specYAML)
		specYAMLETag = fmt.Sprintf("%q", hex.EncodeToString(hYAML[:]))
	}

	// Mount all endpoints onto http.ServeMux using sealed interface dispatch
	for _, ep := range cfg.Endpoints {
		switch h := ep.Handler.(type) {
		case config.OpenAPIHandler:
			e.bindOpenAPIRoute(ep, h, specJSON, specYAML, specJSONETag, specYAMLETag)
		case config.PipelineHandler:
			e.bindPipelineRoute(ep, h)
		}
	}

	return e, nil
}

func (e *Engine) bindOpenAPIRoute(
	ep config.Endpoint,
	h config.OpenAPIHandler,
	specJSON, specYAML []byte,
	specJSONETag, specYAMLETag string,
) {
	e.mux.HandleFunc(ep.MethodAndPath, func(w http.ResponseWriter, r *http.Request) {
		switch h.Mode {
		case "spec":
			isYAML := strings.EqualFold(h.Format, "yaml") || strings.EqualFold(h.Format, "yml")
			contentType := "application/json"
			body := specJSON
			etag := specJSONETag
			if isYAML {
				contentType = "application/yaml"
				body = specYAML
				etag = specYAMLETag
			}

			w.Header().Set("Content-Type", contentType)
			w.Header().Set("ETag", etag)

			if match := r.Header.Get("If-None-Match"); match != "" && (match == etag || match == "*") {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)

		case "ui", "template":
			data := openapi.TemplateData{
				Title:       h.Title,
				Version:     h.Version,
				Description: h.Description,
				SpecURL:     h.SpecURL,
				SpecYAMLURL: strings.TrimSuffix(h.SpecURL, ".json") + ".yaml",
			}

			htmlBytes, err := openapi.RenderHTML(h.Renderer, h.Template, data)
			if err != nil {
				e.errors.Respond(w, r, problem.Problem{
					Type:     e.config.Problem.FormatType("internal-error"),
					Title:    "Documentation Render Error",
					Status:   http.StatusInternalServerError,
					Detail:   err.Error(),
					Instance: r.URL.Path,
				})
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(htmlBytes)
		}
	})
}

func (e *Engine) bindPipelineRoute(ep config.Endpoint, h config.PipelineHandler) {
	var paramNames []string
	matches := pathParamRegex.FindAllStringSubmatch(ep.MethodAndPath, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramNames = append(paramNames, match[1])
		}
	}

	steps := e.buildSteps(h.Steps)
	pipeline := NewPipeline(steps...)

	e.mux.HandleFunc(ep.MethodAndPath, func(w http.ResponseWriter, r *http.Request) {
		execCtx, err := runtime.NewExecutionContext(w, r,
			runtime.WithPathParams(paramNames),
			runtime.WithMaxBodySize(e.config.Server.MaxBodySize.Bytes()),
			runtime.WithProblemTypePrefix(e.config.Problem.TypePrefix),
		)
		if err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				e.errors.Respond(w, r, problem.Problem{
					Type:     e.config.Problem.FormatType("payload-too-large"),
					Title:    "Request Entity Too Large",
					Status:   http.StatusRequestEntityTooLarge,
					Detail:   "request body exceeded maximum allowed size",
					Instance: r.URL.Path,
				})
				return
			}

			// Malformed JSON payload or unreadable request body -> HTTP 400 Bad Request
			e.errors.Respond(w, r, problem.Problem{
				Type:     e.config.Problem.FormatType("bad-request"),
				Title:    "Invalid Request Payload",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
			return
		}

		// Ingress schema validation
		invalidParams := validateRequest(execCtx, ep.RequestRules)
		if len(invalidParams) > 0 {
			e.logger.WarnContext(r.Context(), "request schema validation failed", "path", r.URL.Path, "invalid_count", len(invalidParams))
			e.errors.Respond(w, r, problem.Problem{
				Type:          e.config.Problem.FormatType("validation-error"),
				Title:         "Unprocessable Entity",
				Status:        http.StatusUnprocessableEntity,
				Detail:        "Request payload failed schema validation constraints",
				Instance:      r.URL.Path,
				InvalidParams: invalidParams,
			})
			return
		}

		// Execute pipeline steps
		if err := pipeline.Execute(execCtx, w); err != nil {
			e.errors.Respond(w, r, err)
		}
	})
}

func (e *Engine) buildSteps(parsedSteps []config.ParsedStep) []Step {
	steps := make([]Step, 0, len(parsedSteps))

	for _, ps := range parsedSteps {
		switch ps.Type {
		case config.StepTypeGo:
			steps = append(steps, &GoStep{
				Name:     ps.Name,
				Use:      ps.Go.Use,
				Args:     ps.Go.Args,
				Registry: e.registry,
			})

		case config.StepTypeStarlark:
			steps = append(steps, &StarlarkStep{
				Name:   ps.Name,
				Source: ps.Starlark.Source,
			})

		case config.StepTypeSQL:
			connRef, _ := resolveConnectionRef(ps.SQL.Connection)
			pool, _ := e.sqlManager.Get(connRef)
			steps = append(steps, &SQLStep{
				Name:    ps.Name,
				Pool:    pool,
				Query:   ps.SQL.Query,
				Args:    ps.SQL.Args,
				Catches: ps.SQL.Catches,
			})

		case config.StepTypeRespond:
			steps = append(steps, &RespondStep{
				Condition: ps.Respond.Condition,
				Status:    ps.Respond.Status,
				Headers:   ps.Respond.Headers,
				Body:      ps.Respond.Body,
			})
		}
	}

	return steps
}

// RegisterStep registers a native Go handler callback on the engine.
func (e *Engine) RegisterStep(name string, handler runtime.StepHandler) error {
	return e.registry.Register(name, handler)
}

func (e *Engine) Handler() http.Handler {
	return e.mux
}

func (e *Engine) Server() config.Server {
	return e.config.Server
}

func (e *Engine) Config() *config.Config {
	return e.config
}

func (e *Engine) Close() error {
	if e.sqlManager != nil {
		return e.sqlManager.Close()
	}
	return nil
}

func resolveConnectionRef(expr hcl.Expression) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("missing connection reference expression")
	}
	vars := expr.Variables()
	if len(vars) > 0 {
		var parts []string
		for _, split := range vars[0] {
			switch step := split.(type) {
			case hcl.TraverseRoot:
				parts = append(parts, step.Name)
			case hcl.TraverseAttr:
				parts = append(parts, step.Name)
			}
		}
		return strings.Join(parts, "."), nil
	}
	val, diags := expr.Value(nil)
	if !diags.HasErrors() && val.Type().Equals(cty.String) {
		return val.AsString(), nil
	}
	return "", fmt.Errorf("invalid connection reference expression")
}
