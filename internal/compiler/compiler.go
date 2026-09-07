package compiler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/parser"
)

// OpenAPIMode represents the operational mode of an openapi endpoint handler.
type OpenAPIMode string

const (
	OpenAPIModeSpec     OpenAPIMode = "spec"
	OpenAPIModeUI       OpenAPIMode = "ui"
	OpenAPIModeTemplate OpenAPIMode = "template"
)

// CompiledOpenAPIHandler holds verified configuration for an openapi endpoint.
type CompiledOpenAPIHandler struct {
	Mode        OpenAPIMode
	Format      string
	Renderer    string
	SpecURL     string
	Template    string
	Title       string
	Version     string
	Description string
}

// CompiledRequestRules holds pre-compiled field validation constraints for an endpoint.
type CompiledRequestRules struct {
	PathFields   []manifest.Field
	QueryFields  []manifest.Field
	HeaderFields []manifest.Field
	BodyFields   []manifest.Field
}

// CompiledEndpoint represents a fully verified and compiled HTTP route.
type CompiledEndpoint struct {
	MethodAndPath string
	Description   string
	Steps         []parser.ParsedStep
	Rules         CompiledRequestRules
	OpenAPI       *CompiledOpenAPIHandler
}

// CompiledService represents the entire statically compiled and verified manifest tree.
type CompiledService struct {
	Server      manifest.Server
	OpenAPI     manifest.OpenAPIConfig
	Problem     manifest.ProblemConfig
	Connections []manifest.Connection
	Schemas     map[string][]manifest.Field
	Endpoints   []CompiledEndpoint
}

// Compile performs static semantic analysis on the entire AST manifest tree.
func Compile(m *parser.Manifest, evalCtx *hcl.EvalContext) (*CompiledService, error) {
	if m == nil {
		return &CompiledService{
			Server:  manifest.DefaultServer(),
			OpenAPI: manifest.DefaultOpenAPIConfig(),
			Schemas: make(map[string][]manifest.Field),
		}, nil
	}

	serverConfig, err := m.Server.ToServer()
	if err != nil {
		return nil, fmt.Errorf("server config: %w", err)
	}

	openapiConfig, err := m.OpenAPI.ToOpenAPI(evalCtx)
	if err != nil {
		return nil, fmt.Errorf("openapi config: %w", err)
	}

	problemConfig := m.Problem.ToProblem()

	connections, connIndex, err := compileConnections(m.Connections)
	if err != nil {
		return nil, err
	}

	schemasMap, err := compileSchemas(m.Schemas, evalCtx)
	if err != nil {
		return nil, err
	}

	endpoints, err := compileEndpoints(m.Endpoints, connIndex, schemasMap, openapiConfig, evalCtx)
	if err != nil {
		return nil, err
	}

	if err := resolveSpecURLs(endpoints); err != nil {
		return nil, err
	}

	return &CompiledService{
		Server:      serverConfig,
		OpenAPI:     openapiConfig,
		Problem:     problemConfig,
		Connections: connections,
		Schemas:     schemasMap,
		Endpoints:   endpoints,
	}, nil
}

func compileConnections(blocks []parser.ConnectionBlock) ([]manifest.Connection, map[string]bool, error) {
	var connections []manifest.Connection
	connIndex := make(map[string]bool, len(blocks))

	for _, block := range blocks {
		conn, err := block.ToConnection()
		if err != nil {
			return nil, nil, fmt.Errorf("connection %q: %w", block.Name, err)
		}

		key := conn.Key()
		if connIndex[key] {
			return nil, nil, fmt.Errorf("duplicate connection declaration %q", conn.Reference())
		}

		connIndex[key] = true
		connIndex[conn.Reference()] = true
		connections = append(connections, conn)
	}

	return connections, connIndex, nil
}

func compileSchemas(blocks []parser.SchemaBlock, evalCtx *hcl.EvalContext) (map[string][]manifest.Field, error) {
	schemasMap := make(map[string][]manifest.Field, len(blocks))

	for _, block := range blocks {
		if _, exists := schemasMap[block.Name]; exists {
			return nil, fmt.Errorf("duplicate schema declaration %q", "schema."+block.Name)
		}

		var fields []manifest.Field
		for _, fieldBlock := range block.Fields {
			cf, err := fieldBlock.ToField(evalCtx)
			if err != nil {
				return nil, fmt.Errorf("schema %q: %w", block.Name, err)
			}
			fields = append(fields, cf)
		}
		schemasMap[block.Name] = fields
	}

	return schemasMap, nil
}

// compileEndpoints now simply loops and delegates to compileEndpoint.
func compileEndpoints(
	blocks []parser.EndpointBlock,
	connIndex map[string]bool,
	schemasMap map[string][]manifest.Field,
	openapiConfig manifest.OpenAPIConfig,
	evalCtx *hcl.EvalContext,
) ([]CompiledEndpoint, error) {
	seenRoutes := make(map[string]bool, len(blocks))
	var endpoints []CompiledEndpoint

	for _, ep := range blocks {
		if seenRoutes[ep.MethodAndPath] {
			return nil, fmt.Errorf("duplicate endpoint route %q", ep.MethodAndPath)
		}
		seenRoutes[ep.MethodAndPath] = true

		compiled, err := compileEndpoint(ep, connIndex, schemasMap, openapiConfig, evalCtx)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, compiled)
	}

	return endpoints, nil
}

// compileEndpoint validates handler presence and dispatches between OpenAPI and Pipeline routes.
func compileEndpoint(
	ep parser.EndpointBlock,
	connIndex map[string]bool,
	schemasMap map[string][]manifest.Field,
	openapiConfig manifest.OpenAPIConfig,
	evalCtx *hcl.EvalContext,
) (CompiledEndpoint, error) {
	hasPipeline := ep.Pipeline != nil && ep.Pipeline.Body != nil
	hasOpenAPI := len(ep.OpenAPI) > 0

	// Invariant 1: Single OpenAPI block
	if len(ep.OpenAPI) > 1 {
		modes := make([]string, len(ep.OpenAPI))
		for i, o := range ep.OpenAPI {
			modes[i] = fmt.Sprintf("%q", o.Mode)
		}
		return CompiledEndpoint{}, fmt.Errorf(
			"endpoint %q defines multiple openapi handlers (%s)",
			ep.MethodAndPath,
			strings.Join(modes, ", "),
		)
	}

	// Invariant 2: Mutual exclusion
	if hasPipeline && hasOpenAPI {
		return CompiledEndpoint{}, fmt.Errorf(
			"endpoint %q defines conflicting handlers: 'openapi %q' and 'pipeline'",
			ep.MethodAndPath,
			ep.OpenAPI[0].Mode,
		)
	}

	// Invariant 3: Must declare at least one handler
	if !hasPipeline && !hasOpenAPI {
		return CompiledEndpoint{}, fmt.Errorf("endpoint %q: must declare either a pipeline or an openapi block", ep.MethodAndPath)
	}

	desc := ""
	if ep.Description != nil {
		desc = *ep.Description
	}

	// Branch A: Managed OpenAPI endpoint
	if hasOpenAPI {
		handler, err := compileOpenAPIHandler(ep, openapiConfig)
		if err != nil {
			return CompiledEndpoint{}, err
		}
		return CompiledEndpoint{
			MethodAndPath: ep.MethodAndPath,
			Description:   desc,
			OpenAPI:       handler,
		}, nil
	}

	// Branch B: Dynamic pipeline endpoint
	steps, err := parser.DecodePipelineSteps(ep.Pipeline)
	if err != nil {
		return CompiledEndpoint{}, fmt.Errorf("endpoint %q: %w", ep.MethodAndPath, err)
	}
	if len(steps) == 0 {
		return CompiledEndpoint{}, fmt.Errorf("endpoint %q: pipeline must declare at least one step", ep.MethodAndPath)
	}

	if err := validatePipelineSteps(ep.MethodAndPath, steps, connIndex); err != nil {
		return CompiledEndpoint{}, err
	}

	rules, err := compileRequestRules(ep.MethodAndPath, ep.Request, schemasMap, evalCtx)
	if err != nil {
		return CompiledEndpoint{}, err
	}

	return CompiledEndpoint{
		MethodAndPath: ep.MethodAndPath,
		Description:   desc,
		Steps:         steps,
		Rules:         rules,
	}, nil
}

// compileOpenAPIHandler enforces route method restrictions and delegates mode compilation.
func compileOpenAPIHandler(ep parser.EndpointBlock, openapiConfig manifest.OpenAPIConfig) (*CompiledOpenAPIHandler, error) {
	method, _, err := splitMethodAndPath(ep.MethodAndPath)
	if err != nil {
		return nil, err
	}
	if method != http.MethodGet && method != http.MethodHead {
		return nil, fmt.Errorf("endpoint %q is invalid; openapi endpoints only support HTTP GET and HEAD", ep.MethodAndPath)
	}

	if ep.Request != nil {
		return nil, fmt.Errorf("endpoint %q: openapi endpoints are engine-managed and do not accept a 'request' block", ep.MethodAndPath)
	}

	raw := ep.OpenAPI[0]
	handler := &CompiledOpenAPIHandler{
		Title:       openapiConfig.Title,
		Version:     openapiConfig.Version,
		Description: openapiConfig.Description,
	}
	if raw.SpecURL != nil {
		handler.SpecURL = *raw.SpecURL
	}

	switch OpenAPIMode(raw.Mode) {
	case OpenAPIModeSpec:
		if raw.Renderer != nil || raw.File != nil || raw.Inline != nil {
			return nil, fmt.Errorf("endpoint %q: openapi \"spec\" only accepts the 'format' attribute", ep.MethodAndPath)
		}
		format := "json"
		if raw.Format != nil {
			f := strings.ToLower(strings.TrimSpace(*raw.Format))
			if f != "json" && f != "yaml" && f != "yml" {
				return nil, fmt.Errorf("endpoint %q: invalid openapi format %q; must be 'json' or 'yaml'", ep.MethodAndPath, *raw.Format)
			}
			format = f
		}
		handler.Mode = OpenAPIModeSpec
		handler.Format = format
		return handler, nil

	case OpenAPIModeUI:
		if raw.Format != nil || raw.File != nil || raw.Inline != nil {
			return nil, fmt.Errorf("endpoint %q: openapi \"ui\" only accepts 'renderer' and 'spec_url' attributes", ep.MethodAndPath)
		}
		renderer := "scalar"
		if raw.Renderer != nil {
			r := strings.ToLower(strings.TrimSpace(*raw.Renderer))
			switch r {
			case "scalar", "elements", "swagger", "redoc":
				renderer = r
			default:
				return nil, fmt.Errorf("endpoint %q: unsupported openapi renderer %q; must be 'scalar', 'elements', 'swagger', or 'redoc'", ep.MethodAndPath, *raw.Renderer)
			}
		}
		handler.Mode = OpenAPIModeUI
		handler.Renderer = renderer
		return handler, nil

	case OpenAPIModeTemplate:
		if raw.Format != nil || raw.Renderer != nil {
			return nil, fmt.Errorf("endpoint %q: openapi \"template\" only accepts 'file', 'inline', and 'spec_url' attributes", ep.MethodAndPath)
		}
		if (raw.File == nil && raw.Inline == nil) || (raw.File != nil && raw.Inline != nil) {
			return nil, fmt.Errorf("endpoint %q: openapi \"template\" requires exactly one of 'file' or 'inline'", ep.MethodAndPath)
		}

		handler.Mode = OpenAPIModeTemplate
		if raw.Inline != nil {
			handler.Template = *raw.Inline
		} else if raw.File != nil {
			resolvedPath := manifest.ResolveRelativePath(*raw.File, raw.DeclaringDir)
			content, err := os.ReadFile(resolvedPath)
			if err != nil {
				return nil, fmt.Errorf("endpoint %q: read template file %q: %w", ep.MethodAndPath, *raw.File, err)
			}
			handler.Template = string(content)
		}
		return handler, nil

	default:
		return nil, fmt.Errorf("endpoint %q: unsupported openapi mode %q; allowed modes are \"spec\", \"ui\", \"template\"", ep.MethodAndPath, raw.Mode)
	}
}

func resolveSpecURLs(endpoints []CompiledEndpoint) error {
	type specCandidate struct {
		route string
		path  string
	}

	var jsonSpecs []specCandidate
	for _, ep := range endpoints {
		if ep.OpenAPI != nil && ep.OpenAPI.Mode == OpenAPIModeSpec && ep.OpenAPI.Format == "json" {
			_, p, err := splitMethodAndPath(ep.MethodAndPath)
			if err != nil {
				return err
			}
			jsonSpecs = append(jsonSpecs, specCandidate{
				route: ep.MethodAndPath,
				path:  p,
			})
		}
	}

	for i := range endpoints {
		ep := &endpoints[i]
		if ep.OpenAPI == nil || (ep.OpenAPI.Mode != OpenAPIModeUI && ep.OpenAPI.Mode != OpenAPIModeTemplate) {
			continue
		}

		// Explicit spec_url takes precedence
		if ep.OpenAPI.SpecURL != "" {
			continue
		}

		if len(jsonSpecs) == 0 {
			return fmt.Errorf(
				"cannot auto-derive 'spec_url' for endpoint %q: no 'openapi \"spec\"' endpoint with format \"json\" found; declare a spec endpoint or specify 'spec_url' explicitly",
				ep.MethodAndPath,
			)
		}

		if len(jsonSpecs) == 1 {
			ep.OpenAPI.SpecURL = jsonSpecs[0].path
			continue
		}

		// Multiple candidates: attempt longest path prefix match (e.g. /v1/docs -> /v1/openapi.json)
		_, uiPath, _ := splitMethodAndPath(ep.MethodAndPath)
		uiDir := filepath.Dir(uiPath)

		var prefixMatches []specCandidate
		if uiDir != "/" && uiDir != "." {
			for _, s := range jsonSpecs {
				candDir := filepath.Dir(s.path)
				if candDir == uiDir {
					prefixMatches = append(prefixMatches, s)
				}
			}
		}

		if len(prefixMatches) == 1 {
			ep.OpenAPI.SpecURL = prefixMatches[0].path
			continue
		}

		var candidatePaths []string
		for _, s := range jsonSpecs {
			candidatePaths = append(candidatePaths, fmt.Sprintf("%q", s.path))
		}
		return fmt.Errorf(
			"ambiguous 'spec_url' for endpoint %q: multiple JSON spec endpoints found (%s); specify 'spec_url' explicitly",
			ep.MethodAndPath,
			strings.Join(candidatePaths, ", "),
		)
	}

	return nil
}

func validatePipelineSteps(route string, steps []parser.ParsedStep, connIndex map[string]bool) error {
	seenStepNames := make(map[string]bool, len(steps))

	for _, step := range steps {
		if step.Name != "" {
			if seenStepNames[step.Name] {
				return fmt.Errorf("endpoint %q: duplicate step name %q in pipeline", route, step.Name)
			}
			seenStepNames[step.Name] = true
		}

		// Verify SQL connection references exist
		if step.Type == parser.StepTypeSQL && step.SQL != nil {
			connRef, err := parser.ResolveConnectionRef(step.SQL.Connection)
			if err != nil {
				return fmt.Errorf("endpoint %q: step %q connection: %w", route, step.Name, err)
			}

			cleanRef := strings.TrimPrefix(connRef, "connection.")
			if !connIndex[cleanRef] && !connIndex[connRef] {
				return fmt.Errorf("endpoint %q: step %q: unknown connection %q", route, step.Name, connRef)
			}
		}
	}

	return nil
}

func compileRequestRules(
	route string,
	req *parser.RequestBlock,
	schemasMap map[string][]manifest.Field,
	evalCtx *hcl.EvalContext,
) (CompiledRequestRules, error) {
	var rules CompiledRequestRules
	if req == nil {
		return rules, nil
	}

	compileFields := func(blocks []parser.FieldBlock) ([]manifest.Field, error) {
		var fields []manifest.Field
		for _, fieldBlock := range blocks {
			cf, err := fieldBlock.ToField(evalCtx)
			if err != nil {
				return nil, err
			}
			fields = append(fields, cf)
		}
		return fields, nil
	}

	resolveTarget := func(targetName string, inline *parser.FieldGroupBlock, expr hcl.Expression) ([]manifest.Field, error) {
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
	if rules.PathFields, err = resolveTarget("path", req.PathInline, req.PathExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}
	if rules.QueryFields, err = resolveTarget("query", req.QueryInline, req.QueryExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}
	if rules.HeaderFields, err = resolveTarget("headers", req.HeadersInline, req.HeadersExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}
	if rules.BodyFields, err = resolveTarget("body", req.BodyInline, req.BodyExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}

	return rules, nil
}

func splitMethodAndPath(raw string) (string, string, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid route label %q", raw)
	}
	return strings.ToUpper(parts[0]), parts[1], nil
}
