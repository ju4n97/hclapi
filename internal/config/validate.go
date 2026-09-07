package config

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/scalar"
)

// validate converts the raw parsed HCL structures into a verified Config.
func validate(raw rawManifest, evalCtx *hcl.EvalContext) (*Config, error) {
	cfg := &Config{
		Schemas: make(map[string]Schema),
	}

	// Server settings
	if raw.Server != nil {
		cfg.Server.Host = raw.Server.Host
		cfg.Server.Port = raw.Server.Port
		if raw.Server.ReadTimeout != "" {
			d, err := scalar.ParseDuration(raw.Server.ReadTimeout)
			if err != nil {
				return nil, fmt.Errorf("server: invalid read_timeout: %w", err)
			}
			cfg.Server.ReadTimeout = d
		}
		if raw.Server.WriteTimeout != "" {
			d, err := scalar.ParseDuration(raw.Server.WriteTimeout)
			if err != nil {
				return nil, fmt.Errorf("server: invalid write_timeout: %w", err)
			}
			cfg.Server.WriteTimeout = d
		}
		if raw.Server.IdleTimeout != "" {
			d, err := scalar.ParseDuration(raw.Server.IdleTimeout)
			if err != nil {
				return nil, fmt.Errorf("server: invalid idle_timeout: %w", err)
			}
			cfg.Server.IdleTimeout = d
		}
		if raw.Server.MaxBodySize != "" {
			b, err := scalar.ParseByteSize(raw.Server.MaxBodySize)
			if err != nil {
				return nil, fmt.Errorf("server: invalid max_body_size: %w", err)
			}
			cfg.Server.MaxBodySize = b
		}
	}
	cfg.Server.SetDefaults()

	// Problem settings
	if raw.Problem != nil {
		cfg.Problem = *raw.Problem
	}

	// OpenAPI settings
	if raw.OpenAPI != nil {
		cfg.OpenAPI = *raw.OpenAPI
		if raw.OpenAPI.ServersExpr != nil {
			rawServers, err := eval.Any(raw.OpenAPI.ServersExpr, nil)
			if err != nil {
				return nil, fmt.Errorf("openapi servers: %w", err)
			}
			if list, ok := rawServers.([]any); ok {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						srv := OpenAPIServer{}
						if u, ok := m["url"].(string); ok {
							srv.URL = u
						}
						if d, ok := m["description"].(string); ok {
							srv.Description = d
						}
						cfg.OpenAPI.Servers = append(cfg.OpenAPI.Servers, srv)
					}
				}
			}
		}
		if raw.OpenAPI.TagsExpr != nil {
			rawTags, err := eval.Any(raw.OpenAPI.TagsExpr, nil)
			if err != nil {
				return nil, fmt.Errorf("openapi tags: %w", err)
			}
			if list, ok := rawTags.([]any); ok {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						tag := OpenAPITag{}
						if n, ok := m["name"].(string); ok {
							tag.Name = n
						}
						if d, ok := m["description"].(string); ok {
							tag.Description = d
						}
						cfg.OpenAPI.Tags = append(cfg.OpenAPI.Tags, tag)
					}
				}
			}
		}
	}
	cfg.OpenAPI.SetDefaults()

	// Validate connections
	connIndex := make(map[string]bool)
	for _, rawConn := range raw.Connections {
		conn := Connection{
			Driver: rawConn.Driver,
			Name:   rawConn.Name,
			Source: rawConn.Source,
			Pool: PoolTune{
				MaxOpen: rawConn.Pool.MaxOpen,
				MaxIdle: rawConn.Pool.MaxIdle,
			},
		}
		if rawConn.Pool.MaxLifetime != "" {
			d, err := scalar.ParseDuration(rawConn.Pool.MaxLifetime)
			if err != nil {
				return nil, fmt.Errorf("connection %q: invalid max_lifetime: %w", conn.Name, err)
			}
			conn.Pool.MaxLifetime = d
		}
		if rawConn.Pool.IdleTimeout != "" {
			d, err := scalar.ParseDuration(rawConn.Pool.IdleTimeout)
			if err != nil {
				return nil, fmt.Errorf("connection %q: invalid idle_timeout: %w", conn.Name, err)
			}
			conn.Pool.IdleTimeout = d
		}
		conn.SetDefaults()

		if connIndex[conn.Key()] {
			return nil, fmt.Errorf("duplicate connection declaration %q", conn.Reference())
		}
		connIndex[conn.Key()] = true
		connIndex[conn.Reference()] = true
		cfg.Connections = append(cfg.Connections, conn)
	}

	// Validate schemas and evaluate field properties
	for _, s := range raw.Schemas {
		if _, exists := cfg.Schemas[s.Name]; exists {
			return nil, fmt.Errorf("duplicate schema declaration %q", "schema."+s.Name)
		}
		for i := range s.Fields {
			if err := evaluateField(&s.Fields[i], evalCtx); err != nil {
				return nil, fmt.Errorf("schema %q: %w", s.Name, err)
			}
		}
		cfg.Schemas[s.Name] = s
	}

	// Validate endpoints & construct sealed handlers
	seenRoutes := make(map[string]bool)
	for _, ep := range raw.Endpoints {
		if seenRoutes[ep.MethodAndPath] {
			return nil, fmt.Errorf("duplicate endpoint route %q", ep.MethodAndPath)
		}
		seenRoutes[ep.MethodAndPath] = true

		compiled, err := compileEndpoint(ep, connIndex, cfg.Schemas, cfg.OpenAPI, evalCtx)
		if err != nil {
			return nil, err
		}
		cfg.Endpoints = append(cfg.Endpoints, compiled)
	}

	// Auto-derive spec_url for OpenAPI endpoints
	if err := resolveSpecURLs(cfg.Endpoints); err != nil {
		return nil, err
	}

	return cfg, nil
}

func compileEndpoint(
	ep rawEndpoint,
	connIndex map[string]bool,
	schemas map[string]Schema,
	openapiConfig OpenAPI,
	evalCtx *hcl.EvalContext,
) (Endpoint, error) {
	method, path, err := splitMethodAndPath(ep.MethodAndPath)
	if err != nil {
		return Endpoint{}, err
	}

	hasPipeline := ep.Pipeline != nil && ep.Pipeline.Body != nil
	hasOpenAPI := len(ep.OpenAPIs) > 0

	if len(ep.OpenAPIs) > 1 {
		modes := make([]string, len(ep.OpenAPIs))
		for i, o := range ep.OpenAPIs {
			modes[i] = fmt.Sprintf("%q", o.Mode)
		}
		return Endpoint{}, fmt.Errorf("endpoint %q defines multiple openapi handlers (%s)", ep.MethodAndPath, strings.Join(modes, ", "))
	}

	if hasPipeline && hasOpenAPI {
		return Endpoint{}, fmt.Errorf(
			"endpoint %q defines conflicting handlers: 'openapi %q' and 'pipeline'",
			ep.MethodAndPath,
			ep.OpenAPIs[0].Mode,
		)
	}

	if !hasPipeline && !hasOpenAPI {
		return Endpoint{}, fmt.Errorf("endpoint %q: must declare either a pipeline or an openapi block", ep.MethodAndPath)
	}

	desc := ""
	if ep.Description != nil {
		desc = *ep.Description
	}

	// Branch A: OpenAPI handler
	if hasOpenAPI {
		if ep.Request != nil {
			return Endpoint{}, fmt.Errorf(
				"endpoint %q: openapi endpoints are engine-managed and do not accept a 'request' block",
				ep.MethodAndPath,
			)
		}
		if method != http.MethodGet && method != http.MethodHead {
			return Endpoint{}, fmt.Errorf("endpoint %q is invalid; openapi endpoints only support HTTP GET and HEAD", ep.MethodAndPath)
		}

		raw := ep.OpenAPIs[0]
		handler := OpenAPIHandler{
			Title:       openapiConfig.Title,
			Version:     openapiConfig.Version,
			Description: openapiConfig.Description,
		}
		if raw.SpecURL != nil {
			handler.SpecURL = *raw.SpecURL
		}

		switch raw.Mode {
		case "spec":
			if raw.Renderer != nil || raw.File != nil || raw.Inline != nil {
				return Endpoint{}, fmt.Errorf("endpoint %q: openapi \"spec\" only accepts the 'format' attribute", ep.MethodAndPath)
			}
			format := "json"
			if raw.Format != nil {
				f := strings.ToLower(strings.TrimSpace(*raw.Format))
				if f != "json" && f != "yaml" && f != "yml" {
					return Endpoint{}, fmt.Errorf(
						"endpoint %q: invalid openapi format %q; must be 'json' or 'yaml'",
						ep.MethodAndPath,
						*raw.Format,
					)
				}
				format = f
			}
			handler.Mode = "spec"
			handler.Format = format

		case "ui":
			if raw.Format != nil || raw.File != nil || raw.Inline != nil {
				return Endpoint{}, fmt.Errorf(
					"endpoint %q: openapi \"ui\" only accepts 'renderer' and 'spec_url' attributes",
					ep.MethodAndPath,
				)
			}
			renderer := "scalar"
			if raw.Renderer != nil {
				r := strings.ToLower(strings.TrimSpace(*raw.Renderer))
				switch r {
				case "scalar", "elements", "swagger", "redoc":
					renderer = r
				default:
					return Endpoint{}, fmt.Errorf(
						"endpoint %q: unsupported openapi renderer %q; must be 'scalar', 'elements', 'swagger', or 'redoc'",
						ep.MethodAndPath,
						*raw.Renderer,
					)
				}
			}
			handler.Mode = "ui"
			handler.Renderer = renderer

		case "template":
			if raw.Format != nil || raw.Renderer != nil {
				return Endpoint{}, fmt.Errorf(
					"endpoint %q: openapi \"template\" only accepts 'file', 'inline', and 'spec_url' attributes",
					ep.MethodAndPath,
				)
			}
			if (raw.File == nil && raw.Inline == nil) || (raw.File != nil && raw.Inline != nil) {
				return Endpoint{}, fmt.Errorf(
					"endpoint %q: openapi \"template\" requires exactly one of 'file' or 'inline'",
					ep.MethodAndPath,
				)
			}
			handler.Mode = "template"
			if raw.Inline != nil {
				handler.Template = *raw.Inline
			} else if raw.File != nil {
				resolvedPath := resolveRelativePath(*raw.File, ep.DeclaringDir)
				content, err := os.ReadFile(resolvedPath)
				if err != nil {
					return Endpoint{}, fmt.Errorf("endpoint %q: read template file %q: %w", ep.MethodAndPath, *raw.File, err)
				}
				handler.Template = string(content)
			}

		default:
			return Endpoint{}, fmt.Errorf(
				"endpoint %q: unsupported openapi mode %q; allowed modes are \"spec\", \"ui\", \"template\"",
				ep.MethodAndPath,
				raw.Mode,
			)
		}

		return Endpoint{
			MethodAndPath: ep.MethodAndPath,
			Method:        method,
			Path:          path,
			Description:   desc,
			Handler:       handler,
		}, nil
	}

	// Branch B: Pipeline handler
	steps, err := decodePipelineSteps(ep.Pipeline)
	if err != nil {
		return Endpoint{}, fmt.Errorf("endpoint %q: %w", ep.MethodAndPath, err)
	}
	if len(steps) == 0 {
		return Endpoint{}, fmt.Errorf("endpoint %q: pipeline must declare at least one step", ep.MethodAndPath)
	}

	seenStepNames := make(map[string]bool)
	for _, step := range steps {
		if step.Name != "" {
			if seenStepNames[step.Name] {
				return Endpoint{}, fmt.Errorf("endpoint %q: duplicate step name %q in pipeline", ep.MethodAndPath, step.Name)
			}
			seenStepNames[step.Name] = true
		}
		if step.Type == StepTypeSQL && step.SQL != nil {
			connRef, err := resolveConnectionRef(step.SQL.Connection)
			if err != nil {
				return Endpoint{}, fmt.Errorf("endpoint %q: step %q connection: %w", ep.MethodAndPath, step.Name, err)
			}
			cleanRef := strings.TrimPrefix(connRef, "connection.")
			if !connIndex[cleanRef] && !connIndex[connRef] {
				return Endpoint{}, fmt.Errorf("endpoint %q: step %q: unknown connection %q", ep.MethodAndPath, step.Name, connRef)
			}
		}
	}

	rules, err := compileRequestRules(ep.MethodAndPath, ep.Request, schemas, evalCtx)
	if err != nil {
		return Endpoint{}, err
	}

	return Endpoint{
		MethodAndPath: ep.MethodAndPath,
		Method:        method,
		Path:          path,
		Description:   desc,
		RequestRules:  rules,
		Handler:       PipelineHandler{Steps: steps},
	}, nil
}

func decodePipelineSteps(pipeline *rawPipeline) ([]ParsedStep, error) {
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: string(StepTypeGo), LabelNames: []string{"name"}},
			{Type: string(StepTypeStarlark), LabelNames: []string{"name"}},
			{Type: string(StepTypeSQL), LabelNames: []string{"name"}},
			{Type: string(StepTypeRespond)},
		},
	}

	content, diags := pipeline.Body.Content(schema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decode pipeline: %s", diags.Error())
	}

	var steps []ParsedStep
	for _, block := range content.Blocks {
		switch block.Type {
		case string(StepTypeGo):
			var cfg GoStep
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("go step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{Type: StepTypeGo, Name: block.Labels[0], Go: &cfg})
		case string(StepTypeStarlark):
			var cfg StarlarkStep
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("starlark step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{Type: StepTypeStarlark, Name: block.Labels[0], Starlark: &cfg})
		case string(StepTypeSQL):
			var cfg SQLStep
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("sql step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{Type: StepTypeSQL, Name: block.Labels[0], SQL: &cfg})
		case string(StepTypeRespond):
			var cfg RespondStep
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("respond step: %s", diags.Error())
			}
			steps = append(steps, ParsedStep{Type: StepTypeRespond, Respond: &cfg})
		default:
			return nil, fmt.Errorf("unknown step type %q", block.Type)
		}
	}
	return steps, nil
}

func evaluateField(f *Field, evalCtx *hcl.EvalContext) error {
	f.Type = "any"
	if f.TypeExpr != nil {
		typeVal, err := eval.Any(f.TypeExpr, nil)
		if err != nil {
			return fmt.Errorf("field %q type: %w", f.Name, err)
		}
		if typeVal != nil {
			f.Type = fmt.Sprintf("%v", typeVal)
		}
	}
	if f.EnumExpr != nil {
		rawEnum, err := eval.Any(f.EnumExpr, nil)
		if err != nil {
			return fmt.Errorf("field %q enum: %w", f.Name, err)
		}
		if list, ok := rawEnum.([]any); ok {
			f.Enum = list
		}
	}
	if f.DefaultExpr != nil {
		rawDefault, err := eval.Any(f.DefaultExpr, nil)
		if err != nil {
			return fmt.Errorf("field %q default: %w", f.Name, err)
		}
		f.Default = rawDefault
	}
	return nil
}

func compileRequestRules(route string, req *rawRequest, schemas map[string]Schema, evalCtx *hcl.EvalContext) (RequestRules, error) {
	var rules RequestRules
	if req == nil {
		return rules, nil
	}

	compileFields := func(fields []Field) ([]Field, error) {
		res := make([]Field, len(fields))
		for i := range fields {
			res[i] = fields[i]
			if err := evaluateField(&res[i], evalCtx); err != nil {
				return nil, err
			}
		}
		return res, nil
	}

	resolveTarget := func(target string, inline *rawFieldGroup, expr hcl.Expression) ([]Field, error) {
		if inline != nil {
			return compileFields(inline.Fields)
		}
		if expr != nil {
			ref, err := resolveSchemaRef(expr)
			if err != nil {
				return nil, fmt.Errorf("%s schema: %w", target, err)
			}
			s, exists := schemas[ref]
			if !exists {
				return nil, fmt.Errorf("unknown schema reference %q", "schema."+ref)
			}
			return s.Fields, nil
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

func resolveSpecURLs(endpoints []Endpoint) error {
	type specCandidate struct {
		route string
		path  string
	}

	var jsonSpecs []specCandidate
	for _, ep := range endpoints {
		if h, ok := ep.Handler.(OpenAPIHandler); ok && h.Mode == "spec" && h.Format == "json" {
			jsonSpecs = append(jsonSpecs, specCandidate{route: ep.MethodAndPath, path: ep.Path})
		}
	}

	for i := range endpoints {
		ep := &endpoints[i]
		h, ok := ep.Handler.(OpenAPIHandler)
		if !ok || (h.Mode != "ui" && h.Mode != "template") {
			continue
		}
		if h.SpecURL != "" {
			continue
		}

		if len(jsonSpecs) == 0 {
			return fmt.Errorf(
				"cannot auto-derive 'spec_url' for endpoint %q: no 'openapi \"spec\"' endpoint with format \"json\" found; declare a spec endpoint or specify 'spec_url' explicitly",
				ep.MethodAndPath,
			)
		}

		if len(jsonSpecs) == 1 {
			h.SpecURL = jsonSpecs[0].path
			ep.Handler = h
			continue
		}

		uiDir := filepath.Dir(ep.Path)
		var prefixMatches []specCandidate
		if uiDir != "/" && uiDir != "." {
			for _, s := range jsonSpecs {
				if filepath.Dir(s.path) == uiDir {
					prefixMatches = append(prefixMatches, s)
				}
			}
		}

		if len(prefixMatches) == 1 {
			h.SpecURL = prefixMatches[0].path
			ep.Handler = h
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

func splitMethodAndPath(raw string) (string, string, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid route label %q; expected 'METHOD /path'", raw)
	}
	return strings.ToUpper(parts[0]), parts[1], nil
}

func resolveConnectionRef(expr hcl.Expression) (string, error) {
	if expr == nil {
		return "", errors.New("missing connection reference expression")
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
	return "", errors.New("invalid connection reference expression")
}

func resolveSchemaRef(expr hcl.Expression) (string, error) {
	if expr == nil {
		return "", errors.New("missing schema reference expression")
	}
	vars := expr.Variables()
	if len(vars) > 0 {
		for _, split := range vars[0] {
			if attr, ok := split.(hcl.TraverseAttr); ok {
				return attr.Name, nil
			}
		}
		if root, ok := vars[0][0].(hcl.TraverseRoot); ok && len(vars[0]) == 1 {
			return root.Name, nil
		}
	}
	val, diags := expr.Value(nil)
	if !diags.HasErrors() && val.IsKnown() && !val.IsNull() && val.Type().Equals(cty.String) {
		return strings.TrimPrefix(val.AsString(), "schema."), nil
	}
	return "", errors.New("invalid schema reference expression")
}
