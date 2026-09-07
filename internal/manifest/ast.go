package manifest

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// Manifest represents the unvalidated abstract syntax tree parsed from HCL files.
type Manifest struct {
	Server      *ServerBlock
	OpenAPI     *OpenAPIBlock
	Problem     *ProblemBlock
	Connections []ConnectionBlock
	Schemas     []SchemaBlock
	Endpoints   []EndpointBlock
}

// ServerBlock defines raw listener and transport settings from an HCL server block.
type ServerBlock struct {
	Host         string `hcl:"host,optional"`
	Port         int    `hcl:"port,optional"`
	ReadTimeout  string `hcl:"read_timeout,optional"`
	WriteTimeout string `hcl:"write_timeout,optional"`
	IdleTimeout  string `hcl:"idle_timeout,optional"`
	MaxBodySize  string `hcl:"max_body_size,optional"`
}

// ProblemBlock defines RFC 9457 error type configuration from an HCL problem block.
type ProblemBlock struct {
	TypePrefix string `hcl:"type_prefix,optional"`
}

// OpenAPIBlock defines document-level OpenAPI metadata from an HCL openapi block.
type OpenAPIBlock struct {
	Title       string         `hcl:"title,optional"`
	Version     string         `hcl:"version,optional"`
	Description string         `hcl:"description,optional"`
	ServersExpr hcl.Expression `hcl:"servers,optional"`
	TagsExpr    hcl.Expression `hcl:"tags,optional"`
	Contact     *ContactBlock  `hcl:"contact,block"`
	License     *LicenseBlock  `hcl:"license,block"`
}

// ContactBlock captures maintainer contact information from an HCL contact block.
type ContactBlock struct {
	Name  string `hcl:"name,optional"`
	Email string `hcl:"email,optional"`
	URL   string `hcl:"url,optional"`
}

// LicenseBlock captures API license details from an HCL license block.
type LicenseBlock struct {
	Name string `hcl:"name,optional"`
	URL  string `hcl:"url,optional"`
}

// ConnectionBlock defines a database or pool configuration from an HCL connection block.
type ConnectionBlock struct {
	Driver       string     `hcl:"driver,label"`
	Name         string     `hcl:"name,label"`
	Source       string     `hcl:"source,attr"`
	Pool         *PoolBlock `hcl:"pool,block"`
	DeclaringDir string     `json:"-"`
}

// Key returns the short identifier for the connection pool.
func (c ConnectionBlock) Key() string {
	return c.Driver + "." + c.Name
}

// Reference returns the full manifest expression path for the connection pool.
func (c ConnectionBlock) Reference() string {
	return "connection." + c.Driver + "." + c.Name
}

// PoolBlock defines connection pool capacity limits from an HCL pool block.
type PoolBlock struct {
	MaxOpen     int    `hcl:"max_open,optional"`
	MaxIdle     int    `hcl:"max_idle,optional"`
	MaxLifetime string `hcl:"max_lifetime,optional"`
	IdleTimeout string `hcl:"idle_timeout,optional"`
}

// SchemaBlock defines a named validation structure from an HCL schema block.
type SchemaBlock struct {
	Name        string       `hcl:"name,label"`
	Description string       `hcl:"description,optional"`
	Fields      []FieldBlock `hcl:"field,block"`
}

// FieldBlock defines constraint parameters on a property from an HCL field block.
type FieldBlock struct {
	Name        string         `hcl:"name,label"`
	TypeExpr    hcl.Expression `hcl:"type,attr"`
	Required    bool           `hcl:"required,optional"`
	DefaultExpr hcl.Expression `hcl:"default,optional"`
	Description string         `hcl:"description,optional"`
	EnumExpr    hcl.Expression `hcl:"enum,optional"`
	Format      string         `hcl:"format,optional"`
	Pattern     string         `hcl:"pattern,optional"`
	MinLength   *int           `hcl:"min_length,optional"`
	MaxLength   *int           `hcl:"max_length,optional"`
	Min         *float64       `hcl:"min,optional"`
	Max         *float64       `hcl:"max,optional"`
	MinItems    *int           `hcl:"min_items,optional"`
	MaxItems    *int           `hcl:"max_items,optional"`
	UniqueItems bool           `hcl:"unique_items,optional"`
}

// EndpointBlock defines an HTTP route declaration from an HCL endpoint block.
type EndpointBlock struct {
	MethodAndPath string                  `hcl:"name,label"`
	Description   *string                 `hcl:"description,optional"`
	Request       *RequestBlock           `hcl:"request,block"`
	Pipeline      *PipelineBlock          `hcl:"pipeline,block"`
	OpenAPIs      []OpenAPIOperationBlock `hcl:"openapi,block"`
	DeclaringDir  string                  `json:"-"`
}

// OpenAPIOperationBlock defines documentation handlers inside an endpoint block.
type OpenAPIOperationBlock struct {
	Mode     string   `hcl:"mode,label"`
	Renderer *string  `hcl:"renderer,optional"`
	Format   *string  `hcl:"format,optional"`
	SpecURL  *string  `hcl:"spec_url,optional"`
	File     *string  `hcl:"file,optional"`
	Inline   *string  `hcl:"inline,optional"`
	Remain   hcl.Body `hcl:",remain"`
}

// PipelineBlock captures raw pipeline steps while preserving definition order.
type PipelineBlock struct {
	Body hcl.Body `hcl:",remain"`
}

// RequestBlock captures ingress schemas across path, query, headers, and body.
type RequestBlock struct {
	PathInline    *FieldGroup
	PathExpr      hcl.Expression
	QueryInline   *FieldGroup
	QueryExpr     hcl.Expression
	HeadersInline *FieldGroup
	HeadersExpr   hcl.Expression
	BodyInline    *FieldGroup
	BodyExpr      hcl.Expression
	Remain        hcl.Body `hcl:",remain"`
}

// FieldGroup captures nested inline field blocks within request targets.
type FieldGroup struct {
	Fields []FieldBlock `hcl:"field,block"`
	Remain hcl.Body     `hcl:",remain"`
}

// Decode unpacks inline field blocks or schema reference expressions from the request body.
func (r *RequestBlock) Decode(evalCtx *hcl.EvalContext) error {
	if r == nil || r.Remain == nil {
		return nil
	}

	content, _, diags := r.Remain.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "path", Required: false},
			{Name: "query", Required: false},
			{Name: "headers", Required: false},
			{Name: "body", Required: false},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "path"},
			{Type: "query"},
			{Type: "headers"},
			{Type: "body"},
		},
	})
	if diags.HasErrors() {
		return fmt.Errorf("request block: %s", diags.Error())
	}

	if attr, ok := content.Attributes["path"]; ok {
		r.PathExpr = attr.Expr
	}
	if attr, ok := content.Attributes["query"]; ok {
		r.QueryExpr = attr.Expr
	}
	if attr, ok := content.Attributes["headers"]; ok {
		r.HeadersExpr = attr.Expr
	}
	if attr, ok := content.Attributes["body"]; ok {
		r.BodyExpr = attr.Expr
	}

	for _, block := range content.Blocks {
		var inline FieldGroup
		if err := gohcl.DecodeBody(block.Body, evalCtx, &inline); err.HasErrors() {
			return fmt.Errorf("inline %s block: %s", block.Type, err.Error())
		}
		switch block.Type {
		case "path":
			r.PathInline = &inline
		case "query":
			r.QueryInline = &inline
		case "headers":
			r.HeadersInline = &inline
		case "body":
			r.BodyInline = &inline
		}
	}

	return nil
}
