package config

import (
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

// Validate enriches all AST nodes in-place and checks invariant constraints.
func (cfg *Config) Validate(evalCtx *hcl.EvalContext) error {
	// Server settings
	if cfg.Server != nil {
		if cfg.Server.ReadTimeoutRaw != "" {
			d, err := scalar.ParseDuration(cfg.Server.ReadTimeoutRaw)
			if err != nil {
				return fmt.Errorf("server: invalid read_timeout: %w", err)
			}
			cfg.Server.ReadTimeout = d
		}
		if cfg.Server.WriteTimeoutRaw != "" {
			d, err := scalar.ParseDuration(cfg.Server.WriteTimeoutRaw)
			if err != nil {
				return fmt.Errorf("server: invalid write_timeout: %w", err)
			}
			cfg.Server.WriteTimeout = d
		}
		if cfg.Server.IdleTimeoutRaw != "" {
			d, err := scalar.ParseDuration(cfg.Server.IdleTimeoutRaw)
			if err != nil {
				return fmt.Errorf("server: invalid idle_timeout: %w", err)
			}
			cfg.Server.IdleTimeout = d
		}
		if cfg.Server.MaxBodySizeRaw != "" {
			b, err := scalar.ParseByteSize(cfg.Server.MaxBodySizeRaw)
			if err != nil {
				return fmt.Errorf("server: invalid max_body_size: %w", err)
			}
			cfg.Server.MaxBodySize = b
		}
		cfg.Server.SetDefaults()
	} else {
		var def Server
		def.SetDefaults()
		cfg.Server = &def
	}

	// OpenAPI metadata
	if cfg.OpenAPI != nil {
		if cfg.OpenAPI.ServersExpr != nil {
			raw, err := eval.Any(cfg.OpenAPI.ServersExpr, nil)
			if err != nil {
				return fmt.Errorf("openapi servers: %w", err)
			}
			if list, ok := raw.([]any); ok {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						s := OpenAPIServer{}
						if u, ok := m["url"].(string); ok {
							s.URL = u
						}
						if d, ok := m["description"].(string); ok {
							s.Description = d
						}
						cfg.OpenAPI.Servers = append(cfg.OpenAPI.Servers, s)
					}
				}
			}
		}
		if cfg.OpenAPI.TagsExpr != nil {
			raw, err := eval.Any(cfg.OpenAPI.TagsExpr, nil)
			if err != nil {
				return fmt.Errorf("openapi tags: %w", err)
			}
			if list, ok := raw.([]any); ok {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						t := OpenAPITag{}
						if n, ok := m["name"].(string); ok {
							t.Name = n
						}
						if d, ok := m["description"].(string); ok {
							t.Description = d
						}
						cfg.OpenAPI.Tags = append(cfg.OpenAPI.Tags, t)
					}
				}
			}
		}
		cfg.OpenAPI.SetDefaults()
	} else {
		var def OpenAPI
		def.SetDefaults()
		cfg.OpenAPI = &def
	}

	// Problem settings
	if cfg.Problem == nil {
		cfg.Problem = &Problem{}
	}

	// Validate connections in-place
	connIndex := make(map[string]bool)
	for i := range cfg.Connections {
		conn := &cfg.Connections[i]
		if conn.Pool.MaxLifetimeRaw != "" {
			d, err := scalar.ParseDuration(conn.Pool.MaxLifetimeRaw)
			if err != nil {
				return fmt.Errorf("connection %q: invalid max_lifetime: %w", conn.Name, err)
			}
			conn.Pool.MaxLifetime = d
		}
		if conn.Pool.IdleTimeoutRaw != "" {
			d, err := scalar.ParseDuration(conn.Pool.IdleTimeoutRaw)
			if err != nil {
				return fmt.Errorf("connection %q: invalid idle_timeout: %w", conn.Name, err)
			}
			conn.Pool.IdleTimeout = d
		}
		conn.SetDefaults()

		if connIndex[conn.Key()] {
			return fmt.Errorf("duplicate connection declaration %q", conn.Reference())
		}
		connIndex[conn.Key()] = true
		connIndex[conn.Reference()] = true
	}

	// Index schemas and evaluate fields in-place
	cfg.SchemaMap = make(map[string]Schema, len(cfg.Schemas))
	for i := range cfg.Schemas {
		s := &cfg.Schemas[i]
		if _, exists := cfg.SchemaMap[s.Name]; exists {
			return fmt.Errorf("duplicate schema declaration %q", "schema."+s.Name)
		}
		for j := range s.Fields {
			if err := evaluateField(&s.Fields[j], evalCtx); err != nil {
				return fmt.Errorf("schema %q: %w", s.Name, err)
			}
		}
		cfg.SchemaMap[s.Name] = *s
	}

	// Validate endpoints in-place and construct sealed handlers
	seenRoutes := make(map[string]bool)
	for i := range cfg.Endpoints {
		ep := &cfg.Endpoints[i]
		if seenRoutes[ep.MethodAndPath] {
			return fmt.Errorf("duplicate endpoint route %q", ep.MethodAndPath)
		}
		seenRoutes[ep.MethodAndPath] = true

		if err := ep.enrichAndValidate(connIndex, cfg.SchemaMap, *cfg.OpenAPI, evalCtx); err != nil {
			return err
		}
	}

	// Auto-derive spec_url for OpenAPI endpoints
	return resolveSpecURLs(cfg.Endpoints)
}

func (ep *Endpoint) enrichAndValidate(
	connIndex map[string]bool,
	schemas map[string]Schema,
	openapi OpenAPI,
	evalCtx *hcl.EvalContext,
) error {
	method, path, err := splitMethodAndPath(ep.MethodAndPath)
	if err != nil {
		return err
	}
	ep.Method = method
	ep.Path = path

	hasPipeline := ep.Pipeline != nil && ep.Pipeline.Body != nil
	hasOpenAPI := len(ep.OpenAPIs) > 0

	if len(ep.OpenAPIs) > 1 {
		modes := make([]string, len(ep.OpenAPIs))
		for i, o := range ep.OpenAPIs {
			modes[i] = fmt.Sprintf("%q", o.Mode)
		}
		return fmt.Errorf("endpoint %q defines multiple openapi handlers (%s)", ep.MethodAndPath, strings.Join(modes, ", "))
	}
	if hasPipeline && hasOpenAPI {
		return fmt.Errorf("endpoint %q defines conflicting handlers: 'openapi %q' and 'pipeline'", ep.MethodAndPath, ep.OpenAPIs[0].Mode)
	}
	if !hasPipeline && !hasOpenAPI {
		return fmt.Errorf("endpoint %q: must declare either a pipeline or an openapi block", ep.MethodAndPath)
	}

	// Branch A: OpenAPI handler
	if hasOpenAPI {
		if ep.Request != nil {
			return fmt.Errorf("endpoint %q: openapi endpoints are engine-managed and do not accept a 'request' block", ep.MethodAndPath)
		}
		if method != http.MethodGet && method != http.MethodHead {
			return fmt.Errorf("endpoint %q is invalid; openapi endpoints only support HTTP GET and HEAD", ep.MethodAndPath)
		}

		raw := ep.OpenAPIs[0]
		h := OpenAPIHandler{
			Title:       openapi.Title,
			Version:     openapi.Version,
			Description: openapi.Description,
		}
		if raw.SpecURL != nil {
			h.SpecURL = *raw.SpecURL
		}

		switch raw.Mode {
		case "spec":
			if raw.Renderer != nil || raw.File != nil || raw.Inline != nil {
				return fmt.Errorf("endpoint %q: openapi \"spec\" only accepts the 'format' attribute", ep.MethodAndPath)
			}
			format := "json"
			if raw.Format != nil {
				f := strings.ToLower(strings.TrimSpace(*raw.Format))
				if f != "json" && f != "yaml" && f != "yml" {
					return fmt.Errorf("endpoint %q: invalid openapi format %q; must be 'json' or 'yaml'", ep.MethodAndPath, *raw.Format)
				}
				format = f
			}
			h.Mode = "spec"
			h.Format = format

		case "ui":
			if raw.Format != nil || raw.File != nil || raw.Inline != nil {
				return fmt.Errorf("endpoint %q: openapi \"ui\" only accepts 'renderer' and 'spec_url' attributes", ep.MethodAndPath)
			}
			renderer := "scalar"
			if raw.Renderer != nil {
				r := strings.ToLower(strings.TrimSpace(*raw.Renderer))
				switch r {
				case "scalar", "elements", "swagger", "redoc":
					renderer = r
				default:
					return fmt.Errorf("endpoint %q: unsupported openapi renderer %q; must be 'scalar', 'elements', 'swagger', or 'redoc'", ep.MethodAndPath, *raw.Renderer)
				}
			}
			h.Mode = "ui"
			h.Renderer = renderer

		case "template":
			if raw.Format != nil || raw.Renderer != nil {
				return fmt.Errorf("endpoint %q: openapi \"template\" only accepts 'file', 'inline', and 'spec_url' attributes", ep.MethodAndPath)
			}
			if (raw.File == nil && raw.Inline == nil) || (raw.File != nil && raw.Inline != nil) {
				return fmt.Errorf("endpoint %q: openapi \"template\" requires exactly one of 'file' or 'inline'", ep.MethodAndPath)
			}
			h.Mode = "template"
			if raw.Inline != nil {
				h.Template = *raw.Inline
			} else if raw.File != nil {
				content, err := os.ReadFile(resolveRelativePath(*raw.File, ep.DeclaringDir))
				if err != nil {
					return fmt.Errorf("endpoint %q: read template file %q: %w", ep.MethodAndPath, *raw.File, err)
				}
				h.Template = string(content)
			}

		default:
			return fmt.Errorf("endpoint %q: unsupported openapi mode %q; allowed modes are \"spec\", \"ui\", \"template\"", ep.MethodAndPath, raw.Mode)
		}

		ep.Handler = h
		return nil
	}

	// Branch B: Pipeline handler
	steps, err := decodePipelineSteps(ep.Pipeline)
	if err != nil {
		return fmt.Errorf("endpoint %q: %w", ep.MethodAndPath, err)
	}
	if len(steps) == 0 {
		return fmt.Errorf("endpoint %q: pipeline must declare at least one step", ep.MethodAndPath)
	}

	seenStepNames := make(map[string]bool)
	for _, s := range steps {
		if s.Name != "" {
			if seenStepNames[s.Name] {
				return fmt.Errorf("endpoint %q: duplicate step name %q in pipeline", ep.MethodAndPath, s.Name)
			}
			seenStepNames[s.Name] = true
		}
		if s.Type == StepTypeSQL && s.SQL != nil {
			connRef, err := resolveConnectionRef(s.SQL.Connection)
			if err != nil {
				return fmt.Errorf("endpoint %q: step %q connection: %w", ep.MethodAndPath, s.Name, err)
			}
			cleanRef := strings.TrimPrefix(connRef, "connection.")
			if !connIndex[cleanRef] && !connIndex[connRef] {
				return fmt.Errorf("endpoint %q: step %q: unknown connection %q", ep.MethodAndPath, s.Name, connRef)
			}
		}
	}

	rules, err := compileRequestRules(ep.MethodAndPath, ep.Request, schemas, evalCtx)
	if err != nil {
		return err
	}

	ep.RequestRules = rules
	ep.Handler = PipelineHandler{Steps: steps}
	return nil
}

func decodePipelineSteps(pipeline *PipelineBlock) ([]ParsedStep, error) {
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
			var cfg struct {
				Connection hcl.Expression `hcl:"connection,attr"`
				Query      string         `hcl:"query,attr"`
				Args       hcl.Expression `hcl:"args,optional"`
				Catches    []struct {
					Code    string         `hcl:"code,label"`
					Status  hcl.Expression `hcl:"status,optional"`
					Headers hcl.Expression `hcl:"headers,optional"`
					Body    hcl.Expression `hcl:"body,optional"`
				} `hcl:"catch,block"`
			}
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("sql step %q: %s", block.Labels[0], diags.Error())
			}

			catches := make([]SQLCatch, len(cfg.Catches))
			for i, c := range cfg.Catches {
				catches[i] = SQLCatch{Code: c.Code, Status: c.Status, Headers: c.Headers, Body: c.Body}
			}
			steps = append(steps, ParsedStep{
				Type: StepTypeSQL,
				Name: block.Labels[0],
				SQL: &SQLStep{
					Connection: cfg.Connection,
					Query:      cfg.Query,
					Args:       cfg.Args,
					Catches:    catches,
				},
			})

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
		val, err := eval.Any(f.TypeExpr, nil)
		if err != nil {
			return fmt.Errorf("field %q type: %w", f.Name, err)
		}
		if val != nil {
			f.Type = fmt.Sprintf("%v", val)
		}
	}
	if f.EnumExpr != nil {
		val, err := eval.Any(f.EnumExpr, nil)
		if err != nil {
			return fmt.Errorf("field %q enum: %w", f.Name, err)
		}
		if list, ok := val.([]any); ok {
			f.Enum = list
		}
	}
	if f.DefaultExpr != nil {
		val, err := eval.Any(f.DefaultExpr, nil)
		if err != nil {
			return fmt.Errorf("field %q default: %w", f.Name, err)
		}
		f.Default = val
	}
	return nil
}

func compileRequestRules(route string, req *RequestBlock, schemas map[string]Schema, evalCtx *hcl.EvalContext) (RequestRules, error) {
	var rules RequestRules
	if req == nil {
		return rules, nil
	}

	compileFields := func(fields []Field) ([]Field, error) {
		for i := range fields {
			if err := evaluateField(&fields[i], evalCtx); err != nil {
				return nil, err
			}
		}
		return fields, nil
	}

	resolveTarget := func(target string, inline *FieldGroup, expr hcl.Expression) ([]Field, error) {
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
			return compileFields(s.Fields)
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
			return fmt.Errorf("cannot auto-derive 'spec_url' for endpoint %q: no 'openapi \"spec\"' endpoint with format \"json\" found; declare a spec endpoint or specify 'spec_url' explicitly", ep.MethodAndPath)
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
		return fmt.Errorf("ambiguous 'spec_url' for endpoint %q: multiple JSON spec endpoints found (%s); specify 'spec_url' explicitly", ep.MethodAndPath, strings.Join(candidatePaths, ", "))
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

func resolveSchemaRef(expr hcl.Expression) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("missing schema reference expression")
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
	return "", fmt.Errorf("invalid schema reference expression")
}
