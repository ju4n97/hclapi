// Package engine provides route binding, HTTP multiplexing, and pipeline initialization.
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

	"github.com/ju4n97/hclapi/internal/compiler"
	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/openapi"
	"github.com/ju4n97/hclapi/internal/parser"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
	"github.com/ju4n97/hclapi/internal/validator"
)

var pathParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)(?:\.{3})?\}`)

// Engine is the central HTTP coordinator.
type Engine struct {
	options      manifest.Options
	server       manifest.Server
	openapi      manifest.OpenAPIConfig
	problem      manifest.ProblemConfig
	mux          *http.ServeMux
	sqlManager   *connsql.Manager
	goSteps      map[string]runtime.StepHandler
	errorHandler problem.Handler
	logger       *slog.Logger
}

// New initializes an Engine by parsing manifests, statically compiling services, and registering routes.
func New(options manifest.Options) (*Engine, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	errorHandler := options.ProblemHandler
	if errorHandler == nil {
		errorHandler = problem.DefaultHandler
	}

	evalCtx := eval.BaseContext()
	manifest, err := parser.Parse(options.ConfigPath, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("parse manifests: %w", err)
	}

	// Static compilation and reference verification pass
	service, err := compiler.Compile(manifest, evalCtx)
	if err != nil {
		return nil, err
	}

	bootCtx := context.Background()

	// Initialize database connection pools for compiled connections
	sqlManager := connsql.NewManager()

	for _, conn := range service.Connections {
		if connsql.IsSupportedDriver(conn.Driver) {
			if err := sqlManager.Open(bootCtx, conn); err != nil {
				_ = sqlManager.Close()
				return nil, fmt.Errorf("init connection %q: %w", conn.Reference(), err)
			}
			logger.Info("initialized database connection pool", "connection", conn.Reference(), "driver", conn.Driver)
		}
	}

	e := &Engine{
		options:      options,
		server:       service.Server,
		mux:          http.NewServeMux(),
		sqlManager:   sqlManager,
		goSteps:      make(map[string]runtime.StepHandler),
		errorHandler: errorHandler,
		logger:       logger,
	}

	var (
		specJSON     []byte
		specYAML     []byte
		specJSONETag string
		specYAMLETag string
	)

	hasOpenAPI := false
	for _, endpoint := range service.Endpoints {
		if endpoint.OpenAPI != nil {
			hasOpenAPI = true
			break
		}
	}
	if hasOpenAPI {
		specJSON, err = openapi.GenerateJSON(service, true)
		if err != nil {
			return nil, fmt.Errorf("generate OpenAPI 3.1 JSON: %w", err)
		}
		specYAML, err = openapi.GenerateYAML(service)
		if err != nil {
			return nil, fmt.Errorf("generate OpenAPI 3.1 YAML: %w", err)
		}

		hJSON := sha256.Sum256(specJSON)
		specJSONETag = fmt.Sprintf("%q", hex.EncodeToString(hJSON[:]))

		hYAML := sha256.Sum256(specYAML)
		specYAMLETag = fmt.Sprintf("%q", hex.EncodeToString(hYAML[:]))
	}

	for _, endpoint := range service.Endpoints {
		if endpoint.OpenAPI != nil {
			switch endpoint.OpenAPI.Mode {
			case compiler.OpenAPIModeSpec:
				logger.Info("mounted openapi specification", "route", endpoint.MethodAndPath, "format", endpoint.OpenAPI.Format)
			case compiler.OpenAPIModeUI:
				logger.Info("mounted interactive documentation", "route", endpoint.MethodAndPath, "renderer", endpoint.OpenAPI.Renderer)
			case compiler.OpenAPIModeTemplate:
				logger.Info("mounted custom documentation template", "route", endpoint.MethodAndPath)
			}
			e.bindOpenAPIRoute(endpoint, specJSON, specYAML, specJSONETag, specYAMLETag)
		} else {
			e.bindRoute(endpoint)
		}
	}

	return e, nil
}

func (e *Engine) bindOpenAPIRoute(
	endpoint compiler.CompiledEndpoint,
	specJSON, specYAML []byte,
	specJSONETag, specYAMLETag string,
) {
	e.mux.HandleFunc(endpoint.MethodAndPath, func(w http.ResponseWriter, r *http.Request) {
		handler := endpoint.OpenAPI

		switch handler.Mode {
		case compiler.OpenAPIModeSpec:
			isYAML := strings.EqualFold(handler.Format, "yaml") || strings.EqualFold(handler.Format, "yml")
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

			// RFC 7232 conditional evaluation
			if match := r.Header.Get("If-None-Match"); match != "" && (match == etag || match == "*") {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return

		case compiler.OpenAPIModeUI, compiler.OpenAPIModeTemplate:
			specURL := handler.SpecURL
			specYAMLURL := strings.TrimSuffix(specURL, ".json") + ".yaml"
			if strings.HasSuffix(specURL, ".yaml") || strings.HasSuffix(specURL, ".yml") {
				specYAMLURL = specURL
			}

			data := openapi.TemplateData{
				Title:       handler.Title,
				Version:     handler.Version,
				Description: handler.Description,
				SpecURL:     specURL,
				SpecYAMLURL: specYAMLURL,
			}

			htmlBytes, err := openapi.RenderHTML(handler.Renderer, data, handler.Template, "", "")
			if err != nil {
				e.logger.ErrorContext(r.Context(), "failed to render docs", "error", err)
				e.errorHandler(w, r, problem.Problem{
					Type:     e.problem.ProblemType("internal-error"),
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

func (e *Engine) bindRoute(endpoint compiler.CompiledEndpoint) {
	var paramNames []string
	matches := pathParamRegex.FindAllStringSubmatch(endpoint.MethodAndPath, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramNames = append(paramNames, match[1])
		}
	}

	executor := NewPipelineExecutor(endpoint.Steps, e.goSteps, e.sqlManager)

	e.mux.HandleFunc(endpoint.MethodAndPath, func(w http.ResponseWriter, r *http.Request) {
		execCtx, err := runtime.NewExecutionContext(w, r,
			runtime.WithPathParams(paramNames),
			runtime.WithServer(e.server),
		)
		if err != nil {
			if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
				e.logger.WarnContext(r.Context(), "request payload too large", "error", maxBytesErr, "path", r.URL.Path)
				e.errorHandler(w, r, problem.Problem{
					Type:     e.problem.ProblemType("payload-too-large"),
					Title:    "Request Entity Too Large",
					Status:   http.StatusRequestEntityTooLarge,
					Detail:   "request body exceeded maximum size limit of " + e.server.MaxBodySize.String(),
					Instance: r.URL.Path,
				})
				return
			}

			e.logger.WarnContext(r.Context(), "invalid request payload", "error", err, "path", r.URL.Path)
			e.errorHandler(w, r, problem.Problem{
				Type:     e.problem.ProblemType("bad-request"),
				Title:    "Invalid Request Payload",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
			return
		}

		// Ingress schema validation and normalization
		invalidParams := e.validateRequest(execCtx, endpoint.Rules)
		if len(invalidParams) > 0 {
			e.logger.WarnContext(
				r.Context(),
				"request schema validation failed",
				"path",
				r.URL.Path,
				"invalid_count",
				len(invalidParams),
			)
			e.errorHandler(w, r, problem.Problem{
				Type:          e.problem.ProblemType("validation-error"),
				Title:         "Unprocessable Entity",
				Status:        http.StatusUnprocessableEntity,
				Detail:        "Request payload failed schema validation constraints",
				Instance:      r.URL.Path,
				InvalidParams: invalidParams,
			})
			return
		}

		// Execute pipeline
		if err := executor.Execute(w, execCtx); err != nil {
			if p, ok := errors.AsType[problem.Problem](err); ok {
				if p.Status == 0 {
					p.Status = http.StatusInternalServerError
				}
				if p.Title == "" {
					p.Title = http.StatusText(p.Status)
					if p.Title == "" {
						p.Title = "Error"
					}
				}
				if p.Type == "" {
					p.Type = e.problem.ProblemType(problem.Slugify(p.Title))
				}
				if p.Instance == "" {
					p.Instance = r.URL.Path
				}
				if p.Status >= 500 {
					e.logger.ErrorContext(r.Context(), "step execution failed", "error", p, "path", r.URL.Path)
				} else {
					e.logger.WarnContext(
						r.Context(),
						"step rejected request",
						"status",
						p.Status,
						"title",
						p.Title,
						"path",
						r.URL.Path,
					)
				}

				e.errorHandler(w, r, p)
				return
			}

			e.logger.ErrorContext(r.Context(), "pipeline execution failed", "error", err, "path", r.URL.Path)
			e.errorHandler(w, r, problem.Problem{
				Type:     e.problem.ProblemType("pipeline-execution-failed"),
				Title:    "Pipeline Execution Error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
		}
	})
}

func (e *Engine) validateRequest(execCtx *runtime.ExecutionContext, rules compiler.CompiledRequestRules) []problem.InvalidParam {
	var invalidParams []problem.InvalidParam

	if len(rules.PathFields) > 0 {
		invalidParams = append(invalidParams, validator.ValidateStringMap(execCtx.Request.Path, rules.PathFields)...)
	}

	if len(rules.QueryFields) > 0 {
		invalidParams = append(invalidParams, validator.ValidateStringMap(execCtx.Request.Query, rules.QueryFields)...)
	}

	if len(rules.HeaderFields) > 0 {
		invalidParams = append(invalidParams, validator.ValidateHeaders(execCtx.Request.Headers, rules.HeaderFields)...)
	}

	if len(rules.BodyFields) > 0 {
		bodyMap, ok := execCtx.Request.Body.(map[string]any)
		if !ok {
			if execCtx.Request.Body == nil {
				bodyMap = make(map[string]any)
			} else {
				return append(invalidParams, problem.InvalidParam{
					Name:   "body",
					Reason: "request body must be a JSON object",
				})
			}
		}

		normalizedBody, errs := validator.ValidateBody(bodyMap, rules.BodyFields)
		if len(errs) > 0 {
			invalidParams = append(invalidParams, errs...)
		} else {
			execCtx.Request.Body = normalizedBody
		}
	}

	return invalidParams
}

// Close gracefully closes all active database and cache connection pools.
func (e *Engine) Close() error {
	if e.sqlManager != nil {
		return e.sqlManager.Close()
	}
	return nil
}

// RegisterStep registers a named custom Go function for the pipeline runtime.
func (e *Engine) RegisterStep(name string, handler runtime.StepHandler) error {
	if _, exists := e.goSteps[name]; exists {
		return fmt.Errorf("step %q already registered", name)
	}

	e.goSteps[name] = handler
	return nil
}

// Handler returns the underlying http.Handler multiplexer.
func (e *Engine) Handler() http.Handler {
	return e.mux
}

// Server returns the server configuration with defaults applied.
func (e *Engine) Server() manifest.Server {
	return e.server
}
