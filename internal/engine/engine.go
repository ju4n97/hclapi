// Package engine provides route binding, HTTP multiplexing, and pipeline initialization.
package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/hashicorp/hcl/v2"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
	"github.com/ju4n97/hclapi/internal/validator"
)

// pathParamRegex matches both standard parameters ({id}) and Go 1.22+ catch-all wildcards ({filepath...}).
var pathParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)(?:\.{3})?\}`)

type compiledRequestRules struct {
	pathFields   []core.Field
	queryFields  []core.Field
	headerFields []core.Field
	bodyFields   []core.Field
}

// Engine is the root coordinator managing manifests, step registries, and HTTP routing.
type Engine struct {
	options      core.Options
	server       core.Server
	mux          *http.ServeMux
	sqlManager   *connsql.Manager
	goSteps      map[string]core.StepHandler
	errorHandler core.ErrorHandler
	logger       *slog.Logger
}

// New initializes an Engine by parsing manifests and registering route endpoints.
func New(options core.Options) (*Engine, error) {
	bootCtx := context.Background()

	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	errorHandler := options.ErrorHandler
	if errorHandler == nil {
		errorHandler = core.DefaultErrorHandler
	}

	evalCtx := eval.BaseContext()
	manifest, err := parser.Parse(options.ConfigPath, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("parse manifests: %w", err)
	}

	// Compile standalone schemas
	schemasMap := make(map[string][]core.Field, len(manifest.Schemas))
	for _, schemaBlock := range manifest.Schemas {
		var fields []core.Field
		for _, fieldBlock := range schemaBlock.Fields {
			field, err := fieldBlock.ToField(evalCtx)
			if err != nil {
				return nil, fmt.Errorf("schema %q: %w", schemaBlock.Name, err)
			}
			fields = append(fields, field)
		}
		schemasMap[schemaBlock.Name] = fields
	}

	sqlManager := connsql.NewManager()
	for _, connBlock := range manifest.Connections {
		conn, err := connBlock.ToConnection()
		if err != nil {
			return nil, fmt.Errorf("connection config %q: %w", connBlock.Name, err)
		}

		if connsql.IsSupportedDriver(conn.Driver) {
			if err := sqlManager.Open(bootCtx, conn); err != nil {
				_ = sqlManager.Close()
				return nil, fmt.Errorf("init connection %q: %w", conn.Reference(), err)
			}
			logger.Info("initialized database connection pool", "connection", conn.Reference(), "driver", conn.Driver)
		}
	}

	serverConfig, err := manifest.Server.ToServer()
	if err != nil {
		_ = sqlManager.Close()
		return nil, fmt.Errorf("server config: %w", err)
	}

	e := &Engine{
		options:      options,
		server:       serverConfig,
		mux:          http.NewServeMux(),
		sqlManager:   sqlManager,
		goSteps:      make(map[string]core.StepHandler),
		errorHandler: errorHandler,
		logger:       logger,
	}

	// Compile endpoints and bind routes
	for _, ep := range manifest.Endpoints {
		steps, err := parser.DecodePipelineSteps(&ep.Pipeline)
		if err != nil {
			_ = sqlManager.Close()
			return nil, fmt.Errorf("endpoint %q: %w", ep.MethodAndPath, err)
		}

		rules, err := e.compileRequestRules(ep.Request, schemasMap, evalCtx)
		if err != nil {
			_ = sqlManager.Close()
			return nil, fmt.Errorf("endpoint %q: %w", ep.MethodAndPath, err)
		}

		e.bindRoute(ep.MethodAndPath, steps, rules)
	}

	return e, nil
}

func (e *Engine) compileRequestRules(
	req *parser.RequestBlock,
	schemasMap map[string][]core.Field,
	evalCtx *hcl.EvalContext,
) (compiledRequestRules, error) {
	var rules compiledRequestRules
	if req == nil {
		return rules, nil
	}

	compileFields := func(blocks []parser.FieldBlock) ([]core.Field, error) {
		var fields []core.Field
		for _, fieldBlock := range blocks {
			field, err := fieldBlock.ToField(evalCtx)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		}
		return fields, nil
	}

	resolveTarget := func(targetName string, inline *parser.FieldGroupBlock, expr hcl.Expression) ([]core.Field, error) {
		if inline != nil {
			return compileFields(inline.Fields)
		}
		if expr != nil {
			schemaRef, err := parser.ResolveSchemaRef(expr)
			if err != nil {
				return nil, fmt.Errorf("%s schema: %w", targetName, err)
			}
			fields, exists := schemasMap[schemaRef]
			if !exists {
				return nil, fmt.Errorf("unknown schema reference %q", "schema."+schemaRef)
			}
			return fields, nil
		}
		return nil, nil
	}

	var err error
	if rules.pathFields, err = resolveTarget("path", req.PathInline, req.PathExpr); err != nil {
		return rules, err
	}
	if rules.queryFields, err = resolveTarget("query", req.QueryInline, req.QueryExpr); err != nil {
		return rules, err
	}
	if rules.headerFields, err = resolveTarget("headers", req.HeadersInline, req.HeadersExpr); err != nil {
		return rules, err
	}
	if rules.bodyFields, err = resolveTarget("body", req.BodyInline, req.BodyExpr); err != nil {
		return rules, err
	}

	return rules, nil
}

func (e *Engine) bindRoute(routePattern string, steps []parser.ParsedStep, rules compiledRequestRules) {
	var paramNames []string
	matches := pathParamRegex.FindAllStringSubmatch(routePattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			paramNames = append(paramNames, match[1])
		}
	}

	executor := NewPipelineExecutor(steps, e.goSteps, e.sqlManager)

	e.mux.HandleFunc(routePattern, func(w http.ResponseWriter, r *http.Request) {
		hclapiCtx, err := core.NewContext(w, r,
			core.WithPathParams(paramNames),
			core.WithServer(e.server),
		)
		if err != nil {
			if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
				e.logger.WarnContext(r.Context(), "request payload too large", "error", maxBytesErr, "path", r.URL.Path)
				e.errorHandler(w, r, core.ProblemDetailsError{
					Type:     e.server.ProblemType("payload-too-large"),
					Title:    "Request Entity Too Large",
					Status:   http.StatusRequestEntityTooLarge,
					Detail:   "request body exceeded maximum size limit of " + e.server.MaxBodySize.String(),
					Instance: r.URL.Path,
				})
				return
			}

			e.logger.WarnContext(r.Context(), "invalid request payload", "error", err, "path", r.URL.Path)
			e.errorHandler(w, r, core.ProblemDetailsError{
				Type:     e.server.ProblemType("bad-request"),
				Title:    "Invalid Request Payload",
				Status:   http.StatusBadRequest,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
			return
		}

		invalidParams := e.validateRequest(hclapiCtx, rules)
		if len(invalidParams) > 0 {
			e.logger.WarnContext(
				r.Context(),
				"request schema validation failed",
				"path",
				r.URL.Path,
				"invalid_count",
				len(invalidParams),
			)
			e.errorHandler(w, r, core.ProblemDetailsError{
				Type:          e.server.ProblemType("validation-error"),
				Title:         "Unprocessable Entity",
				Status:        http.StatusUnprocessableEntity,
				Detail:        "Request payload failed schema validation constraints",
				Instance:      r.URL.Path,
				InvalidParams: invalidParams,
			})
			return
		}

		if err := executor.Execute(w, hclapiCtx); err != nil {
			e.logger.ErrorContext(r.Context(), "pipeline execution failed", "error", err, "path", r.URL.Path)
			e.errorHandler(w, r, core.ProblemDetailsError{
				Type:     e.server.ProblemType("pipeline-execution-failed"),
				Title:    "Pipeline Execution Error",
				Status:   http.StatusInternalServerError,
				Detail:   err.Error(),
				Instance: r.URL.Path,
			})
		}
	})
}

func (e *Engine) validateRequest(ctx *core.Context, rules compiledRequestRules) []core.InvalidParam {
	var invalidParams []core.InvalidParam

	// Validate and normalize path parameters
	if len(rules.pathFields) > 0 {
		if errs := validator.ValidateStringMap(ctx.Request.Path, rules.pathFields); len(errs) > 0 {
			invalidParams = append(invalidParams, errs...)
		} else {
			validator.NormalizeStringMap(ctx.Request.Path, rules.pathFields)
		}
	}

	// Validate and normalize query parameters
	if len(rules.queryFields) > 0 {
		if errs := validator.ValidateStringMap(ctx.Request.Query, rules.queryFields); len(errs) > 0 {
			invalidParams = append(invalidParams, errs...)
		} else {
			validator.NormalizeStringMap(ctx.Request.Query, rules.queryFields)
		}
	}

	// Validate headers
	if len(rules.headerFields) > 0 {
		if errs := validator.ValidateStringMap(ctx.Request.Headers, rules.headerFields); len(errs) > 0 {
			invalidParams = append(invalidParams, errs...)
		}
	}

	// Validate and normalize body
	if len(rules.bodyFields) > 0 {
		bodyMap, ok := ctx.Request.Body.(map[string]any)
		if !ok {
			if ctx.Request.Body == nil {
				bodyMap = make(map[string]any)
			} else {
				invalidParams = append(invalidParams, core.InvalidParam{
					Name:   "body",
					Reason: "request body must be a JSON object",
				})
			}
		}

		if ok || ctx.Request.Body == nil {
			if errs := validator.Validate(bodyMap, rules.bodyFields); len(errs) > 0 {
				invalidParams = append(invalidParams, errs...)
			} else {
				ctx.Request.Body = validator.Normalize(bodyMap, rules.bodyFields)
			}
		}
	}

	return invalidParams
}

// Close gracefully closes all active connection pools.
func (e *Engine) Close() error {
	if e.sqlManager != nil {
		return e.sqlManager.Close()
	}
	return nil
}

// RegisterStep registers a named custom Go function for the pipeline runtime.
func (e *Engine) RegisterStep(name string, handler core.StepHandler) error {
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
func (e *Engine) Server() core.Server {
	return e.server
}
