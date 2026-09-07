package service

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/scalar"
)

// Build lowers a raw HCL Manifest into an immutable, fully validated service definition.
func Build(m *manifest.Manifest, evalCtx *hcl.EvalContext) (*Definition, error) {
	if m == nil {
		return nil, fmt.Errorf("cannot build service from nil manifest")
	}

	def := &Definition{
		Schemas: make(map[string]Schema, len(m.Schemas)),
	}

	server, err := lowerServer(m.Server)
	if err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}
	def.Server = server

	if m.Problem != nil {
		def.Problem = Problem{TypePrefix: m.Problem.TypePrefix}
	}

	openapi, err := lowerOpenAPI(m.OpenAPI, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("openapi: %w", err)
	}
	def.OpenAPI = openapi

	connIndex := make(map[string]bool)
	for _, rawConn := range m.Connections {
		conn, err := lowerConnection(rawConn)
		if err != nil {
			return nil, err
		}
		if connIndex[conn.Key()] {
			return nil, fmt.Errorf("duplicate connection declaration %q", conn.Reference())
		}
		connIndex[conn.Key()] = true
		connIndex[conn.Reference()] = true
		def.Connections = append(def.Connections, conn)
	}

	for _, rawSchema := range m.Schemas {
		if _, exists := def.Schemas[rawSchema.Name]; exists {
			return nil, fmt.Errorf("duplicate schema declaration %q", "schema."+rawSchema.Name)
		}
		s, err := lowerSchema(rawSchema, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("schema %q: %w", rawSchema.Name, err)
		}
		def.Schemas[s.Name] = s
	}

	seenRoutes := make(map[string]bool)
	for _, rawEp := range m.Endpoints {
		if seenRoutes[rawEp.MethodAndPath] {
			return nil, fmt.Errorf("duplicate endpoint route %q", rawEp.MethodAndPath)
		}
		seenRoutes[rawEp.MethodAndPath] = true

		ep, err := lowerEndpoint(rawEp, connIndex, def.Schemas, def.OpenAPI, evalCtx)
		if err != nil {
			return nil, err
		}
		def.Endpoints = append(def.Endpoints, ep)
	}

	if err := resolveSpecURLs(def.Endpoints); err != nil {
		return nil, err
	}

	return def, nil
}

func lowerServer(b *manifest.ServerBlock) (Server, error) {
	s := Server{
		Host:         "127.0.0.1",
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		MaxBodySize:  10 * 1024 * 1024,
	}
	if b == nil {
		return s, nil
	}

	if b.Host != "" {
		s.Host = b.Host
	}
	if b.Port != 0 {
		s.Port = b.Port
	}
	if b.ReadTimeout != "" {
		d, err := scalar.ParseDuration(b.ReadTimeout)
		if err != nil {
			return s, fmt.Errorf("read_timeout: %w", err)
		}
		s.ReadTimeout = d.Duration()
	}
	if b.WriteTimeout != "" {
		d, err := scalar.ParseDuration(b.WriteTimeout)
		if err != nil {
			return s, fmt.Errorf("write_timeout: %w", err)
		}
		s.WriteTimeout = d.Duration()
	}
	if b.IdleTimeout != "" {
		d, err := scalar.ParseDuration(b.IdleTimeout)
		if err != nil {
			return s, fmt.Errorf("idle_timeout: %w", err)
		}
		s.IdleTimeout = d.Duration()
	}
	if b.MaxBodySize != "" {
		bs, err := scalar.ParseByteSize(b.MaxBodySize)
		if err != nil {
			return s, fmt.Errorf("max_body_size: %w", err)
		}
		s.MaxBodySize = bs.Bytes()
	}

	return s, nil
}

func lowerOpenAPI(b *manifest.OpenAPIBlock, evalCtx *hcl.EvalContext) (OpenAPI, error) {
	o := OpenAPI{
		Title:   "API Documentation",
		Version: "1.0.0",
	}
	if b == nil {
		return o, nil
	}

	if b.Title != "" {
		o.Title = b.Title
	}
	if b.Version != "" {
		o.Version = b.Version
	}
	if b.Description != "" {
		o.Description = b.Description
	}

	if b.Contact != nil {
		o.Contact = &Contact{
			Name:  b.Contact.Name,
			Email: b.Contact.Email,
			URL:   b.Contact.URL,
		}
	}
	if b.License != nil {
		o.License = &License{
			Name: b.License.Name,
			URL:  b.License.URL,
		}
	}

	if b.ServersExpr != nil {
		raw, err := eval.Any(b.ServersExpr, nil)
		if err != nil {
			return o, fmt.Errorf("servers: %w", err)
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
					o.Servers = append(o.Servers, s)
				}
			}
		}
	}

	if b.TagsExpr != nil {
		raw, err := eval.Any(b.TagsExpr, nil)
		if err != nil {
			return o, fmt.Errorf("tags: %w", err)
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
					o.Tags = append(o.Tags, t)
				}
			}
		}
	}

	return o, nil
}

func lowerConnection(b manifest.ConnectionBlock) (Connection, error) {
	c := Connection{
		Driver: b.Driver,
		Name:   b.Name,
		Source: b.Source,
		Pool: PoolConfig{
			MaxOpen:     25,
			MaxIdle:     5,
			MaxLifetime: 30 * time.Minute,
			IdleTimeout: 5 * time.Minute,
		},
	}

	if b.Pool != nil {
		if b.Pool.MaxOpen != 0 {
			c.Pool.MaxOpen = b.Pool.MaxOpen
		}
		if b.Pool.MaxIdle != 0 {
			c.Pool.MaxIdle = b.Pool.MaxIdle
		}
		if b.Pool.MaxLifetime != "" {
			d, err := scalar.ParseDuration(b.Pool.MaxLifetime)
			if err != nil {
				return c, fmt.Errorf("connection %q: max_lifetime: %w", c.Name, err)
			}
			c.Pool.MaxLifetime = d.Duration()
		}
		if b.Pool.IdleTimeout != "" {
			d, err := scalar.ParseDuration(b.Pool.IdleTimeout)
			if err != nil {
				return c, fmt.Errorf("connection %q: idle_timeout: %w", c.Name, err)
			}
			c.Pool.IdleTimeout = d.Duration()
		}
	}

	return c, nil
}

func lowerSchema(b manifest.SchemaBlock, evalCtx *hcl.EvalContext) (Schema, error) {
	s := Schema{
		Name:        b.Name,
		Description: b.Description,
		Fields:      make([]Field, len(b.Fields)),
	}

	for i, fb := range b.Fields {
		f, err := lowerField(fb, evalCtx)
		if err != nil {
			return s, err
		}
		s.Fields[i] = f
	}

	return s, nil
}

func lowerField(b manifest.FieldBlock, evalCtx *hcl.EvalContext) (Field, error) {
	f := Field{
		Name:        b.Name,
		Type:        "any",
		Required:    b.Required,
		Description: b.Description,
		Format:      b.Format,
		Pattern:     b.Pattern,
		MinLength:   b.MinLength,
		MaxLength:   b.MaxLength,
		Min:         b.Min,
		Max:         b.Max,
		MinItems:    b.MinItems,
		MaxItems:    b.MaxItems,
		UniqueItems: b.UniqueItems,
	}

	if b.TypeExpr != nil {
		val, err := eval.Any(b.TypeExpr, nil)
		if err != nil {
			return f, fmt.Errorf("field %q type: %w", b.Name, err)
		}
		if val != nil {
			f.Type = fmt.Sprintf("%v", val)
		}
	}

	if b.EnumExpr != nil {
		val, err := eval.Any(b.EnumExpr, nil)
		if err != nil {
			return f, fmt.Errorf("field %q enum: %w", b.Name, err)
		}
		if list, ok := val.([]any); ok {
			f.Enum = list
		}
	}

	if b.DefaultExpr != nil {
		val, err := eval.Any(b.DefaultExpr, nil)
		if err != nil {
			return f, fmt.Errorf("field %q default: %w", b.Name, err)
		}
		f.Default = val
	}

	return f, nil
}

func lowerEndpoint(
	b manifest.EndpointBlock,
	connIndex map[string]bool,
	schemas map[string]Schema,
	openapi OpenAPI,
	evalCtx *hcl.EvalContext,
) (Endpoint, error) {
	method, path, err := splitMethodAndPath(b.MethodAndPath)
	if err != nil {
		return Endpoint{}, err
	}

	ep := Endpoint{
		Method:       method,
		Path:         path,
		RoutePattern: b.MethodAndPath,
	}
	if b.Description != nil {
		ep.Description = *b.Description
	}

	hasPipeline := b.Pipeline != nil && b.Pipeline.Body != nil
	hasOpenAPI := len(b.OpenAPIs) > 0

	if len(b.OpenAPIs) > 1 {
		modes := make([]string, len(b.OpenAPIs))
		for i, o := range b.OpenAPIs {
			modes[i] = fmt.Sprintf("%q", o.Mode)
		}
		return ep, fmt.Errorf("endpoint %q defines multiple openapi handlers (%s)", b.MethodAndPath, strings.Join(modes, ", "))
	}
	if hasPipeline && hasOpenAPI {
		return ep, fmt.Errorf("endpoint %q defines conflicting handlers: 'openapi %q' and 'pipeline'", b.MethodAndPath, b.OpenAPIs[0].Mode)
	}
	if !hasPipeline && !hasOpenAPI {
		return ep, fmt.Errorf("endpoint %q: must declare either a pipeline or an openapi block", b.MethodAndPath)
	}

	if hasOpenAPI {
		if b.Request != nil {
			return ep, fmt.Errorf("endpoint %q: openapi endpoints are engine-managed and do not accept a 'request' block", b.MethodAndPath)
		}
		if method != http.MethodGet && method != http.MethodHead {
			return ep, fmt.Errorf("endpoint %q is invalid; openapi endpoints only support HTTP GET and HEAD", b.MethodAndPath)
		}

		raw := b.OpenAPIs[0]
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
				return ep, fmt.Errorf("endpoint %q: openapi \"spec\" only accepts the 'format' attribute", b.MethodAndPath)
			}
			format := "json"
			if raw.Format != nil {
				f := strings.ToLower(strings.TrimSpace(*raw.Format))
				if f != "json" && f != "yaml" && f != "yml" {
					return ep, fmt.Errorf("endpoint %q: invalid openapi format %q; must be 'json' or 'yaml'", b.MethodAndPath, *raw.Format)
				}
				format = f
			}
			h.Mode = "spec"
			h.Format = format

		case "ui":
			if raw.Format != nil || raw.File != nil || raw.Inline != nil {
				return ep, fmt.Errorf("endpoint %q: openapi \"ui\" only accepts 'renderer' and 'spec_url' attributes", b.MethodAndPath)
			}
			renderer := "scalar"
			if raw.Renderer != nil {
				r := strings.ToLower(strings.TrimSpace(*raw.Renderer))
				switch r {
				case "scalar", "elements", "swagger", "redoc":
					renderer = r
				default:
					return ep, fmt.Errorf("endpoint %q: unsupported openapi renderer %q; must be 'scalar', 'elements', 'swagger', or 'redoc'", b.MethodAndPath, *raw.Renderer)
				}
			}
			h.Mode = "ui"
			h.Renderer = renderer

		case "template":
			if raw.Format != nil || raw.Renderer != nil {
				return ep, fmt.Errorf("endpoint %q: openapi \"template\" only accepts 'file', 'inline', and 'spec_url' attributes", b.MethodAndPath)
			}
			if (raw.File == nil && raw.Inline == nil) || (raw.File != nil && raw.Inline != nil) {
				return ep, fmt.Errorf("endpoint %q: openapi \"template\" requires exactly one of 'file' or 'inline'", b.MethodAndPath)
			}
			h.Mode = "template"
			if raw.Inline != nil {
				h.Template = *raw.Inline
			} else if raw.File != nil {
				targetFile := *raw.File
				if !filepath.IsAbs(targetFile) && b.DeclaringDir != "" {
					targetFile = filepath.Join(b.DeclaringDir, targetFile)
				}
				content, err := os.ReadFile(targetFile)
				if err != nil {
					return ep, fmt.Errorf("endpoint %q: read template file %q: %w", b.MethodAndPath, *raw.File, err)
				}
				h.Template = string(content)
			}

		default:
			return ep, fmt.Errorf("endpoint %q: unsupported openapi mode %q; allowed modes are \"spec\", \"ui\", \"template\"", b.MethodAndPath, raw.Mode)
		}

		ep.Handler = h
		return ep, nil
	}

	steps, err := decodePipelineSteps(b.Pipeline)
	if err != nil {
		return ep, fmt.Errorf("endpoint %q: %w", b.MethodAndPath, err)
	}
	if len(steps) == 0 {
		return ep, fmt.Errorf("endpoint %q: pipeline must declare at least one step", b.MethodAndPath)
	}

	seenStepNames := make(map[string]bool)
	for _, s := range steps {
		if s.Name != "" {
			if seenStepNames[s.Name] {
				return ep, fmt.Errorf("endpoint %q: duplicate step name %q in pipeline", b.MethodAndPath, s.Name)
			}
			seenStepNames[s.Name] = true
		}
		if s.Type == StepTypeSQL && s.SQL != nil {
			cleanKey := strings.TrimPrefix(s.SQL.ConnectionKey, "connection.")
			if !connIndex[cleanKey] && !connIndex[s.SQL.ConnectionKey] {
				return ep, fmt.Errorf("endpoint %q: step %q: unknown connection %q", b.MethodAndPath, s.Name, s.SQL.ConnectionKey)
			}
		}
	}

	rules, err := compileRequestRules(b.MethodAndPath, b.Request, schemas, evalCtx)
	if err != nil {
		return ep, err
	}

	ep.RequestRules = rules
	ep.Handler = PipelineHandler{Steps: steps}
	return ep, nil
}

func decodePipelineSteps(pipeline *manifest.PipelineBlock) ([]Step, error) {
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

	var steps []Step
	for _, block := range content.Blocks {
		switch block.Type {
		case string(StepTypeGo):
			var cfg struct {
				Use  string         `hcl:"use,attr"`
				Args hcl.Expression `hcl:"args,optional"`
			}
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("go step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, Step{
				Type: StepTypeGo,
				Name: block.Labels[0],
				Go:   &GoStep{Use: cfg.Use, Args: cfg.Args},
			})

		case string(StepTypeStarlark):
			var cfg struct {
				Source string `hcl:"source,attr"`
			}
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("starlark step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, Step{
				Type:     StepTypeStarlark,
				Name:     block.Labels[0],
				Starlark: &StarlarkStep{Source: cfg.Source},
			})

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

			connRef, err := resolveConnectionRef(cfg.Connection)
			if err != nil {
				return nil, fmt.Errorf("sql step %q connection: %w", block.Labels[0], err)
			}

			catches := make([]SQLCatch, len(cfg.Catches))
			for i, c := range cfg.Catches {
				catches[i] = SQLCatch{Code: c.Code, Status: c.Status, Headers: c.Headers, Body: c.Body}
			}

			steps = append(steps, Step{
				Type: StepTypeSQL,
				Name: block.Labels[0],
				SQL: &SQLStep{
					ConnectionKey: connRef,
					Query:         cfg.Query,
					Args:          cfg.Args,
					Catches:       catches,
				},
			})

		case string(StepTypeRespond):
			var cfg struct {
				Condition hcl.Expression `hcl:"condition,optional"`
				Status    hcl.Expression `hcl:"status,optional"`
				Headers   hcl.Expression `hcl:"headers,optional"`
				Body      hcl.Expression `hcl:"body,optional"`
			}
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("respond step: %s", diags.Error())
			}
			steps = append(steps, Step{
				Type: StepTypeRespond,
				Respond: &RespondStep{
					Condition: cfg.Condition,
					Status:    cfg.Status,
					Headers:   cfg.Headers,
					Body:      cfg.Body,
				},
			})

		default:
			return nil, fmt.Errorf("unknown step type %q", block.Type)
		}
	}

	return steps, nil
}

func compileRequestRules(
	route string,
	req *manifest.RequestBlock,
	schemas map[string]Schema,
	evalCtx *hcl.EvalContext,
) (RequestRules, error) {
	var rules RequestRules
	if req == nil {
		return rules, nil
	}

	compileFields := func(fieldBlocks []manifest.FieldBlock) ([]Field, error) {
		fields := make([]Field, len(fieldBlocks))
		for i, fb := range fieldBlocks {
			f, err := lowerField(fb, evalCtx)
			if err != nil {
				return nil, err
			}
			fields[i] = f
		}
		return fields, nil
	}

	resolveTarget := func(target string, inline *manifest.FieldGroup, expr hcl.Expression) ([]Field, error) {
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
			jsonSpecs = append(jsonSpecs, specCandidate{route: ep.RoutePattern, path: ep.Path})
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
			return fmt.Errorf("cannot auto-derive 'spec_url' for endpoint %q: no 'openapi \"spec\"' endpoint with format \"json\" found; declare a spec endpoint or specify 'spec_url' explicitly", ep.RoutePattern)
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
		return fmt.Errorf("ambiguous 'spec_url' for endpoint %q: multiple JSON spec endpoints found (%s); specify 'spec_url' explicitly", ep.RoutePattern, strings.Join(candidatePaths, ", "))
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
