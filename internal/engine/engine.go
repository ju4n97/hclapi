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

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/openapi"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
	"github.com/ju4n97/hclapi/internal/scalar"
	"github.com/ju4n97/hclapi/internal/service"
	"github.com/ju4n97/hclapi/internal/sqldb"
)

var pathParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)(?:\.{3})?\}`)

// Engine coordinates routing, connection pools, and pipeline execution.
type Engine struct {
	options    Options
	service    *service.Definition
	mux        *http.ServeMux
	sqlManager *sqldb.Manager
	registry   *StepRegistry
	errors     *ErrorResponder
	logger     *slog.Logger
}

// New initializes an Engine by loading manifests and lowering them into a service definition.
func New(options Options) (*Engine, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	evalCtx := eval.BaseContext()
	rawManifest, err := manifest.Load(options.ConfigPath, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	svc, err := service.Build(rawManifest, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("build service: %w", err)
	}

	bootCtx := context.Background()
	sqlManager := sqldb.NewManager()

	for _, conn := range svc.Connections {
		if sqldb.IsSupportedDriver(conn.Driver) {
			dbCfg := sqldb.Config{
				Driver: conn.Driver,
				Name:   conn.Name,
				Source: conn.Source,
				Pool: sqldb.PoolConfig{
					MaxOpen:     conn.Pool.MaxOpen,
					MaxIdle:     conn.Pool.MaxIdle,
					MaxLifetime: conn.Pool.MaxLifetime,
					IdleTimeout: conn.Pool.IdleTimeout,
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
	errorResponder := NewErrorResponder(svc.Problem, options.ProblemHandler, logger)

	e := &Engine{
		options:    options,
		service:    svc,
		mux:        http.NewServeMux(),
		sqlManager: sqlManager,
		registry:   registry,
		errors:     errorResponder,
		logger:     logger,
	}

	var (
		specJSON     []byte
		specYAML     []byte
		specJSONETag string
		specYAMLETag string
	)

	hasOpenAPI := false
	for _, endpoint := range svc.Endpoints {
		if _, ok := endpoint.Handler.(service.OpenAPIHandler); ok {
			hasOpenAPI = true
			break
		}
	}

	if hasOpenAPI {
		specJSON, err = openapi.GenerateJSON(svc, true)
		if err != nil {
			return nil, fmt.Errorf("generate OpenAPI 3.1 JSON: %w", err)
		}
		specYAML, err = openapi.GenerateYAML(svc)
		if err != nil {
			return nil, fmt.Errorf("generate OpenAPI 3.1 YAML: %w", err)
		}

		hJSON := sha256.Sum256(specJSON)
		specJSONETag = fmt.Sprintf("%q", hex.EncodeToString(hJSON[:]))
		hYAML := sha256.Sum256(specYAML)
		specYAMLETag = fmt.Sprintf("%q", hex.EncodeToString(hYAML[:]))
	}

	for _, ep := range svc.Endpoints {
		switch h := ep.Handler.(type) {
		case service.OpenAPIHandler:
			e.bindOpenAPIRoute(ep, h, specJSON, specYAML, specJSONETag, specYAMLETag)
		case service.PipelineHandler:
			e.bindPipelineRoute(ep, h)
		}
	}

	return e, nil
}

func (e *Engine) bindOpenAPIRoute(
	ep service.Endpoint,
	h service.OpenAPIHandler,
	specJSON, specYAML []byte,
	specJSONETag, specYAMLETag string,
) {
	e.mux.HandleFunc(ep.RoutePattern, func(w http.ResponseWriter, r *http.Request) {
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
					Type:     e.formatProblemType("internal-error"),
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

func (e *Engine) bindPipelineRoute(ep service.Endpoint, h service.PipelineHandler) {
	var paramNames []string
	matches := pathParamRegex.FindAllStringSubmatch(ep.RoutePattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramNames = append(paramNames, match[1])
		}
	}

	steps := e.buildSteps(h.Steps)
	pipeline := NewPipeline(steps...)

	e.mux.HandleFunc(ep.RoutePattern, func(w http.ResponseWriter, r *http.Request) {
		maxBodySize := e.service.Server.MaxBodySize
		problemPrefix := e.service.Problem.TypePrefix

		execCtx, err := runtime.NewExecutionContext(w, r,
			runtime.WithPathParams(paramNames),
			runtime.WithMaxBodySize(maxBodySize),
			runtime.WithProblemTypePrefix(problemPrefix),
		)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				bodySizeStr := scalar.ByteSize(maxBodySize).String()
				e.errors.Respond(w, r, problem.Problem{
					Type:     e.formatProblemType("payload-too-large"),
					Title:    "Request Entity Too Large",
					Status:   http.StatusRequestEntityTooLarge,
					Detail:   "request body exceeded maximum size limit of " + bodySizeStr,
					Instance: r.URL.Path,
				})
				return
			}

			e.errors.Respond(w, r, problem.Problem{
				Type:     e.formatProblemType("bad-request"),
				Title:    "Invalid Request Payload",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
			return
		}

		invalidParams := validateRequest(execCtx, ep.RequestRules)
		if len(invalidParams) > 0 {
			e.logger.WarnContext(r.Context(), "request schema validation failed", "path", r.URL.Path, "invalid_count", len(invalidParams))
			e.errors.Respond(w, r, problem.Problem{
				Type:          e.formatProblemType("validation-error"),
				Title:         "Unprocessable Entity",
				Status:        http.StatusUnprocessableEntity,
				Detail:        "Request payload failed schema validation constraints",
				Instance:      r.URL.Path,
				InvalidParams: invalidParams,
			})
			return
		}

		if err := pipeline.Execute(execCtx, w); err != nil {
			e.errors.Respond(w, r, err)
		}
	})
}

func (e *Engine) formatProblemType(slug string) string {
	return e.service.Problem.FormatType(slug)
}

func (e *Engine) buildSteps(parsedSteps []service.Step) []Step {
	steps := make([]Step, 0, len(parsedSteps))

	for _, ps := range parsedSteps {
		switch ps.Type {
		case service.StepTypeGo:
			steps = append(steps, &GoStep{
				Name:     ps.Name,
				Use:      ps.Go.Use,
				Args:     ps.Go.Args,
				Registry: e.registry,
			})

		case service.StepTypeStarlark:
			steps = append(steps, &StarlarkStep{
				Name:   ps.Name,
				Source: ps.Starlark.Source,
			})

		case service.StepTypeSQL:
			pool, _ := e.sqlManager.Get(ps.SQL.ConnectionKey)
			steps = append(steps, &SQLStep{
				Name:    ps.Name,
				Pool:    pool,
				Query:   ps.SQL.Query,
				Args:    ps.SQL.Args,
				Catches: ps.SQL.Catches,
			})

		case service.StepTypeRespond:
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

// Handler returns the HTTP handler for the engine.
func (e *Engine) Handler() http.Handler {
	return e.mux
}

// Server returns a copy of the resolved server transport configuration.
func (e *Engine) Server() service.Server {
	return e.service.Server
}

// Service returns a copy of the active service definition.
func (e *Engine) Service() *service.Definition {
	return e.service
}

// Close closes all active database connection pools.
func (e *Engine) Close() error {
	if e.sqlManager != nil {
		return e.sqlManager.Close()
	}
	return nil
}
