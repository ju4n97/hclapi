package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/hcl/v2"
	"gopkg.in/yaml.v3"

	"github.com/ju4n97/hclapi/internal/config"
	"github.com/ju4n97/hclapi/internal/eval"
)

// Generate builds a validated OpenAPI 3.1.0 document from a validated *config.Config.
func Generate(cfg *config.Config) (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:       cfg.OpenAPI.Title,
			Version:     cfg.OpenAPI.Version,
			Description: cfg.OpenAPI.Description,
		},
		Paths:      openapi3.NewPaths(),
		Components: &openapi3.Components{Schemas: make(openapi3.Schemas)},
	}

	if cfg.OpenAPI.Contact != nil {
		doc.Info.Contact = &openapi3.Contact{
			Name:  cfg.OpenAPI.Contact.Name,
			Email: cfg.OpenAPI.Contact.Email,
			URL:   cfg.OpenAPI.Contact.URL,
		}
	}

	if cfg.OpenAPI.License != nil {
		doc.Info.License = &openapi3.License{
			Name: cfg.OpenAPI.License.Name,
			URL:  cfg.OpenAPI.License.URL,
		}
	}

	for _, srv := range cfg.OpenAPI.Servers {
		doc.Servers = append(doc.Servers, &openapi3.Server{
			URL:         srv.URL,
			Description: srv.Description,
		})
	}

	for _, tag := range cfg.OpenAPI.Tags {
		doc.Tags = append(doc.Tags, &openapi3.Tag{
			Name:        tag.Name,
			Description: tag.Description,
		})
	}

	// 1. Map reusable schema components
	for schemaName, schema := range cfg.Schemas {
		schemaObj, err := fieldsToObjectSchema(schema.Fields)
		if err != nil {
			return nil, fmt.Errorf("schema %q: %w", schemaName, err)
		}
		doc.Components.Schemas[schemaName] = &openapi3.SchemaRef{Value: schemaObj}
	}

	// 2. Map API endpoints (excluding documentation UI and spec routes)
	for _, endpoint := range cfg.Endpoints {
		if _, isDocs := endpoint.Handler.(config.OpenAPIHandler); isDocs {
			continue // Exclude documentation and spec routes from the operations catalog
		}

		openapiPath := convertToOpenAPIPath(endpoint.Path)
		op, err := buildOperation(endpoint, cfg)
		if err != nil {
			return nil, fmt.Errorf("endpoint %q: %w", endpoint.MethodAndPath, err)
		}

		pathItem := doc.Paths.Find(openapiPath)
		if pathItem == nil {
			pathItem = &openapi3.PathItem{}
			doc.Paths.Set(openapiPath, pathItem)
		}
		pathItem.SetOperation(endpoint.Method, op)
	}

	// Validate spec adherence
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate generated openapi spec: %w", err)
	}

	return doc, nil
}

func buildOperation(ep config.Endpoint, cfg *config.Config) (*openapi3.Operation, error) {
	op := openapi3.NewOperation()
	if ep.Description != "" {
		op.Summary = ep.Description
		op.Description = ep.Description
	}

	tag := deriveTag(ep.Path)
	if tag != "" {
		op.Tags = []string{tag}
	}

	for _, f := range ep.RequestRules.PathFields {
		s, err := fieldToSchema(f)
		if err != nil {
			return nil, err
		}
		param := openapi3.NewPathParameter(f.Name).WithSchema(s)
		param.Required = true
		if f.Description != "" {
			param.Description = f.Description
		}
		op.AddParameter(param)
	}

	for _, f := range ep.RequestRules.QueryFields {
		s, err := fieldToSchema(f)
		if err != nil {
			return nil, err
		}
		param := openapi3.NewQueryParameter(f.Name).WithSchema(s)
		param.Required = f.Required
		if f.Description != "" {
			param.Description = f.Description
		}
		op.AddParameter(param)
	}

	for _, f := range ep.RequestRules.HeaderFields {
		s, err := fieldToSchema(f)
		if err != nil {
			return nil, err
		}
		param := openapi3.NewHeaderParameter(f.Name).WithSchema(s)
		param.Required = f.Required
		if f.Description != "" {
			param.Description = f.Description
		}
		op.AddParameter(param)
	}

	if len(ep.RequestRules.BodyFields) > 0 {
		schema, err := fieldsToObjectSchema(ep.RequestRules.BodyFields)
		if err != nil {
			return nil, err
		}
		reqBody := openapi3.NewRequestBody().
			WithJSONSchema(schema).
			WithRequired(true)
		op.RequestBody = &openapi3.RequestBodyRef{Value: reqBody}
	}

	statusCodes := make(map[int]bool)
	if pipeline, ok := ep.Handler.(config.PipelineHandler); ok {
		for _, step := range pipeline.Steps {
			if step.Type == config.StepTypeRespond && step.Respond != nil {
				if code := evaluateStaticStatus(step.Respond.Status); code > 0 {
					statusCodes[code] = true
				}
			}
			if step.Type == config.StepTypeSQL && step.SQL != nil {
				for _, c := range step.SQL.Catches {
					if code := evaluateStaticStatus(c.Status); code > 0 {
						statusCodes[code] = true
					}
				}
			}
		}
	}

	if len(statusCodes) == 0 {
		statusCodes[http.StatusOK] = true
	}
	if len(ep.RequestRules.PathFields) > 0 || len(ep.RequestRules.QueryFields) > 0 ||
		len(ep.RequestRules.HeaderFields) > 0 || len(ep.RequestRules.BodyFields) > 0 {
		statusCodes[http.StatusUnprocessableEntity] = true
	}
	if cfg.Server.MaxBodySize > 0 {
		statusCodes[http.StatusRequestEntityTooLarge] = true
	}
	statusCodes[http.StatusInternalServerError] = true

	for code := range statusCodes {
		statusText := http.StatusText(code)
		if statusText == "" {
			statusText = "Response"
		}
		op.AddResponse(code, openapi3.NewResponse().WithDescription(statusText))
	}

	return op, nil
}

func fieldToSchema(f config.Field) (*openapi3.Schema, error) {
	schema := &openapi3.Schema{}

	switch {
	case f.Type == "string":
		schema.Type = &openapi3.Types{openapi3.TypeString}
		if f.MinLength != nil {
			schema.MinLength = uint64(*f.MinLength)
		}
		if f.MaxLength != nil {
			v := uint64(*f.MaxLength)
			schema.MaxLength = &v
		}
		if f.Pattern != "" {
			schema.Pattern = f.Pattern
		}
		if f.Format != "" {
			schema.Format = f.Format
		}

	case f.Type == "int":
		schema.Type = &openapi3.Types{openapi3.TypeInteger}
		if f.Min != nil {
			schema.Min = f.Min
		}
		if f.Max != nil {
			schema.Max = f.Max
		}

	case f.Type == "float":
		schema.Type = &openapi3.Types{openapi3.TypeNumber}
		if f.Min != nil {
			schema.Min = f.Min
		}
		if f.Max != nil {
			schema.Max = f.Max
		}

	case f.Type == "bool":
		schema.Type = &openapi3.Types{openapi3.TypeBoolean}

	case strings.HasPrefix(f.Type, "list"):
		schema.Type = &openapi3.Types{openapi3.TypeArray}
		elemType := strings.TrimSuffix(strings.TrimPrefix(f.Type, "list("), ")")
		if elemType != "" && elemType != f.Type {
			subSchema, err := fieldToSchema(config.Field{Type: elemType})
			if err != nil {
				return nil, err
			}
			schema.Items = &openapi3.SchemaRef{Value: subSchema}
		}
		if f.MinItems != nil {
			schema.MinItems = uint64(*f.MinItems)
		}
		if f.MaxItems != nil {
			v := uint64(*f.MaxItems)
			schema.MaxItems = &v
		}
		schema.UniqueItems = f.UniqueItems

	case strings.HasPrefix(f.Type, "map"):
		schema.Type = &openapi3.Types{openapi3.TypeObject}
		elemType := strings.TrimSuffix(strings.TrimPrefix(f.Type, "map("), ")")
		if elemType != "" && elemType != f.Type && elemType != "any" {
			valSchema, err := fieldToSchema(config.Field{Type: elemType})
			if err != nil {
				return nil, err
			}
			schema.AdditionalProperties = openapi3.AdditionalProperties{
				Schema: &openapi3.SchemaRef{Value: valSchema},
			}
		}

	case f.Type == "any":
		return &openapi3.Schema{}, nil

	default:
		return nil, fmt.Errorf("unsupported schema field type %q", f.Type)
	}

	if len(f.Enum) > 0 {
		schema.Enum = f.Enum
	}
	if f.Default != nil {
		schema.Default = f.Default
	}
	if f.Description != "" {
		schema.Description = f.Description
	}

	return schema, nil
}

func fieldsToObjectSchema(fields []config.Field) (*openapi3.Schema, error) {
	obj := openapi3.NewObjectSchema()
	for _, f := range fields {
		s, err := fieldToSchema(f)
		if err != nil {
			return nil, err
		}
		obj.Properties[f.Name] = &openapi3.SchemaRef{Value: s}
		if f.Required {
			obj.Required = append(obj.Required, f.Name)
		}
	}
	return obj, nil
}

func evaluateStaticStatus(expr hcl.Expression) int {
	if expr == nil {
		return 0
	}
	val, err := eval.Int(expr, nil, 0)
	if err == nil {
		return val
	}
	return 0
}

func convertToOpenAPIPath(p string) string {
	return strings.ReplaceAll(p, "...}", "}")
}

func deriveTag(p string) string {
	trimmed := strings.Trim(p, "/")
	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if !strings.HasPrefix(part, "api") && !strings.HasPrefix(part, "v") && !strings.HasPrefix(part, "{") {
			return part
		}
	}
	return "default"
}

// GenerateJSON serializes the OpenAPI 3.1 specification to formatted JSON.
func GenerateJSON(cfg *config.Config, pretty bool) ([]byte, error) {
	doc, err := Generate(cfg)
	if err != nil {
		return nil, err
	}
	if pretty {
		return json.MarshalIndent(doc, "", "  ")
	}
	return json.Marshal(doc)
}

// GenerateYAML converts the specification into clean YAML.
func GenerateYAML(cfg *config.Config) ([]byte, error) {
	jsonBytes, err := GenerateJSON(cfg, false)
	if err != nil {
		return nil, err
	}

	var parsed any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode openapi json: %w", err)
	}

	return yaml.Marshal(parsed)
}
