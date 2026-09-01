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

	"github.com/ju4n97/hclapi/internal/compiler"
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
)

// Generate builds a validated OpenAPI 3.1.0 document from a CompiledService.
func Generate(service *compiler.CompiledService) (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:       service.Server.OpenAPI.Title,
			Version:     service.Server.OpenAPI.Version,
			Description: service.Server.OpenAPI.Description,
		},
		Paths:      openapi3.NewPaths(),
		Components: &openapi3.Components{Schemas: make(openapi3.Schemas)},
	}

	if service.Server.OpenAPI.Contact != nil {
		doc.Info.Contact = &openapi3.Contact{
			Name:  service.Server.OpenAPI.Contact.Name,
			Email: service.Server.OpenAPI.Contact.Email,
			URL:   service.Server.OpenAPI.Contact.URL,
		}
	}

	if service.Server.OpenAPI.License != nil {
		doc.Info.License = &openapi3.License{
			Name: service.Server.OpenAPI.License.Name,
			URL:  service.Server.OpenAPI.License.URL,
		}
	}

	for _, srv := range service.Server.OpenAPI.Servers {
		doc.Servers = append(doc.Servers, &openapi3.Server{
			URL:         srv.URL,
			Description: srv.Description,
		})
	}

	for _, tag := range service.Server.OpenAPI.Tags {
		doc.Tags = append(doc.Tags, &openapi3.Tag{
			Name:        tag.Name,
			Description: tag.Description,
		})
	}

	// Index standalone reusable schemas
	for schemaName, fields := range service.Schemas {
		schemaObj, err := fieldsToObjectSchema(fields)
		if err != nil {
			return nil, fmt.Errorf("schema %q: %w", schemaName, err)
		}
		doc.Components.Schemas[schemaName] = &openapi3.SchemaRef{Value: schemaObj}
	}

	// Map API Endpoints
	for _, endpoint := range service.Endpoints {
		if endpoint.OpenAPI != nil {
			continue // Skip documentation portal routes from API spec
		}

		method, pathPattern, err := splitMethodAndPath(endpoint.MethodAndPath)
		if err != nil {
			return nil, err
		}

		openapiPath := convertToOpenAPIPath(pathPattern)
		op, err := buildOperation(endpoint, pathPattern)
		if err != nil {
			return nil, fmt.Errorf("endpoint %q: %w", endpoint.MethodAndPath, err)
		}

		pathItem := doc.Paths.Find(openapiPath)
		if pathItem == nil {
			pathItem = &openapi3.PathItem{}
			doc.Paths.Set(openapiPath, pathItem)
		}
		pathItem.SetOperation(method, op)
	}

	// Built-in validation check via kin-openapi
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate generated openapi spec: %w", err)
	}

	return doc, nil
}

func buildOperation(ep compiler.CompiledEndpoint, pathPattern string) (*openapi3.Operation, error) {
	op := openapi3.NewOperation()
	if ep.Description != "" {
		op.Summary = ep.Description
		op.Description = ep.Description
	}

	tag := deriveTag(pathPattern)
	if tag != "" {
		op.Tags = []string{tag}
	}

	// Path Parameters
	for _, f := range ep.Rules.PathFields {
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

	// Query Parameters
	for _, f := range ep.Rules.QueryFields {
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

	// Header Parameters
	for _, f := range ep.Rules.HeaderFields {
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

	// Request Body
	if len(ep.Rules.BodyFields) > 0 {
		schema, err := fieldsToObjectSchema(ep.Rules.BodyFields)
		if err != nil {
			return nil, err
		}
		reqBody := openapi3.NewRequestBody().
			WithJSONSchema(schema).
			WithRequired(true)
		op.RequestBody = &openapi3.RequestBodyRef{Value: reqBody}
	}

	// Responses derived statically from AST expressions
	statusCodes := make(map[int]bool)
	for _, step := range ep.Steps {
		if step.Type == parser.StepTypeRespond && step.Respond != nil {
			if code := evaluateStaticStatus(step.Respond.Status); code > 0 {
				statusCodes[code] = true
			}
		}
		if step.Type == parser.StepTypeSQL && step.SQL != nil {
			for _, c := range step.SQL.Catches {
				if code := evaluateStaticStatus(c.Status); code > 0 {
					statusCodes[code] = true
				}
			}
		}
	}

	if len(statusCodes) == 0 {
		statusCodes[http.StatusOK] = true
	}
	if len(ep.Rules.PathFields) > 0 || len(ep.Rules.QueryFields) > 0 || len(ep.Rules.HeaderFields) > 0 ||
		len(ep.Rules.BodyFields) > 0 {
		statusCodes[http.StatusUnprocessableEntity] = true
	}
	statusCodes[http.StatusInternalServerError] = true

	for code := range statusCodes {
		statusText := http.StatusText(code)
		if statusText == "" {
			statusText = "Response"
		}
		resp := openapi3.NewResponse().WithDescription(statusText)
		op.AddResponse(code, resp)
	}

	return op, nil
}

func fieldToSchema(f core.Field) (*openapi3.Schema, error) {
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
			subSchema, err := fieldToSchema(core.Field{Type: elemType})
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
			valSchema, err := fieldToSchema(core.Field{Type: elemType})
			if err != nil {
				return nil, err
			}
			schema.AdditionalProperties = openapi3.AdditionalProperties{
				Schema: &openapi3.SchemaRef{Value: valSchema},
			}
		}

	case f.Type == "any":
		// Untyped in OpenAPI 3.1 is an empty schema
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

func fieldsToObjectSchema(fields []core.Field) (*openapi3.Schema, error) {
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

func splitMethodAndPath(raw string) (string, string, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid route label %q", raw)
	}
	return strings.ToUpper(parts[0]), parts[1], nil
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
func GenerateJSON(service *compiler.CompiledService, pretty bool) ([]byte, error) {
	doc, err := Generate(service)
	if err != nil {
		return nil, err
	}
	if pretty {
		return json.MarshalIndent(doc, "", "  ")
	}
	return json.Marshal(doc)
}

// GenerateYAML converts the canonical JSON representation into YAML to preserve custom kin-openapi tags.
func GenerateYAML(service *compiler.CompiledService) ([]byte, error) {
	jsonBytes, err := GenerateJSON(service, false)
	if err != nil {
		return nil, err
	}

	var parsed any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode openapi json: %w", err)
	}

	return yaml.Marshal(parsed)
}
