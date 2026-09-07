package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/ju4n97/hclapi/internal/scalar"
)

// Config represents the complete, validated Abstract Syntax Tree of an API service.
type Config struct {
	Server      *Server      `hcl:"server,block"`
	OpenAPI     *OpenAPI     `hcl:"openapi,block"`
	Problem     *Problem     `hcl:"problem,block"`
	Connections []Connection `hcl:"connection,block"`
	Schemas     []Schema     `hcl:"schema,block"`
	Endpoints   []Endpoint   `hcl:"endpoint,block"`
	Remain      hcl.Body     `hcl:",remain"`

	// SchemaMap provides O(1) schema lookups indexed during validation.
	SchemaMap map[string]Schema `json:"-"`
}

// Server defines transport-level listener and socket settings.
type Server struct {
	Host            string `hcl:"host,optional"`
	Port            int    `hcl:"port,optional"`
	ReadTimeoutRaw  string `hcl:"read_timeout,optional"`
	WriteTimeoutRaw string `hcl:"write_timeout,optional"`
	IdleTimeoutRaw  string `hcl:"idle_timeout,optional"`
	MaxBodySizeRaw  string `hcl:"max_body_size,optional"`

	// ReadTimeout is the parsed maximum duration for reading request headers and body.
	ReadTimeout scalar.Duration `json:"-"`
	// WriteTimeout is the parsed maximum duration before timing out response writes.
	WriteTimeout scalar.Duration `json:"-"`
	// IdleTimeout is the parsed maximum duration an idle keep-alive connection remains open.
	IdleTimeout scalar.Duration `json:"-"`
	// MaxBodySize is the parsed upper byte limit on incoming request bodies.
	MaxBodySize scalar.ByteSize `json:"-"`
}

// SetDefaults applies production baseline settings to unset server attributes.
func (s *Server) SetDefaults() {
	if s.Host == "" {
		s.Host = "127.0.0.1"
	}
	if s.Port == 0 {
		s.Port = 8080
	}
	if s.ReadTimeout == 0 {
		s.ReadTimeout = scalar.Duration(15 * time.Second)
	}
	if s.WriteTimeout == 0 {
		s.WriteTimeout = scalar.Duration(15 * time.Second)
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = scalar.Duration(60 * time.Second)
	}
	if s.MaxBodySize == 0 {
		s.MaxBodySize = scalar.ByteSize(10 * 1024 * 1024)
	}
}

// Problem configures RFC 9457 Problem Details error type URI resolution.
type Problem struct {
	TypePrefix string `hcl:"type_prefix,optional"`
}

// FormatType returns the fully qualified error type URI for a given slug.
func (p Problem) FormatType(slug string) string {
	if p.TypePrefix == "" {
		return "urn:hclapi:error:" + slug
	}
	if strings.HasPrefix(p.TypePrefix, "http://") || strings.HasPrefix(p.TypePrefix, "https://") {
		return strings.TrimSuffix(p.TypePrefix, "/") + "/" + slug
	}
	return p.TypePrefix + slug
}

// OpenAPI holds document metadata for the OpenAPI 3.1 specification.
type OpenAPI struct {
	Title       string         `hcl:"title,optional"`
	Version     string         `hcl:"version,optional"`
	Description string         `hcl:"description,optional"`
	ServersExpr hcl.Expression `hcl:"servers,optional"`
	TagsExpr    hcl.Expression `hcl:"tags,optional"`
	Contact     *Contact       `hcl:"contact,block"`
	License     *License       `hcl:"license,block"`

	// Servers holds statically evaluated deployment targets.
	Servers []OpenAPIServer `json:"-"`
	// Tags holds statically evaluated category grouping tags.
	Tags []OpenAPITag `json:"-"`
}

// SetDefaults applies baseline metadata values when omitted from manifests.
func (o *OpenAPI) SetDefaults() {
	if o.Title == "" {
		o.Title = "API Documentation"
	}
	if o.Version == "" {
		o.Version = "1.0.0"
	}
}

// OpenAPIServer defines a deployment target URL in the specification.
type OpenAPIServer struct {
	URL         string
	Description string
}

// OpenAPITag defines a category tag for operation grouping.
type OpenAPITag struct {
	Name        string
	Description string
}

// Contact holds API maintainer contact information.
type Contact struct {
	Name  string `hcl:"name,optional"`
	Email string `hcl:"email,optional"`
	URL   string `hcl:"url,optional"`
}

// License defines legal license information for the API.
type License struct {
	Name string `hcl:"name,optional"`
	URL  string `hcl:"url,optional"`
}

// Connection represents a relational database or cache connection pool definition.
type Connection struct {
	Driver string     `hcl:"driver,label"`
	Name   string     `hcl:"name,label"`
	Source string     `hcl:"source,attr"`
	Pool   PoolConfig `hcl:"pool,block"`
}

// PoolConfig defines sizing and lifecycle policies for a connection pool.
type PoolConfig struct {
	MaxOpen        int    `hcl:"max_open,optional"`
	MaxIdle        int    `hcl:"max_idle,optional"`
	MaxLifetimeRaw string `hcl:"max_lifetime,optional"`
	IdleTimeoutRaw string `hcl:"idle_timeout,optional"`

	// MaxLifetime is the parsed duration a connection may be reused before retirement.
	MaxLifetime scalar.Duration `json:"-"`
	// IdleTimeout is the parsed duration an unused connection remains open before eviction.
	IdleTimeout scalar.Duration `json:"-"`
}

// SetDefaults applies baseline connection pool limits when omitted.
func (c *Connection) SetDefaults() {
	if c.Pool.MaxOpen == 0 {
		c.Pool.MaxOpen = 25
	}
	if c.Pool.MaxIdle == 0 {
		c.Pool.MaxIdle = 5
	}
	if c.Pool.MaxLifetime == 0 {
		c.Pool.MaxLifetime = scalar.Duration(30 * time.Minute)
	}
	if c.Pool.IdleTimeout == 0 {
		c.Pool.IdleTimeout = scalar.Duration(5 * time.Minute)
	}
}

// Key returns the short identifier for the pool (e.g. "postgres.primary").
func (c Connection) Key() string {
	return c.Driver + "." + c.Name
}

// Reference returns the full manifest reference path (e.g. "connection.postgres.primary").
func (c Connection) Reference() string {
	return "connection." + c.Driver + "." + c.Name
}

// Schema represents a reusable request payload validation schema.
type Schema struct {
	Name        string  `hcl:"name,label"`
	Description string  `hcl:"description,optional"`
	Fields      []Field `hcl:"field,block"`
}

// Field defines validation constraints on an input parameter or payload property.
type Field struct {
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

	// Type is the evaluated canonical type identifier ("string", "int", "list(string)", etc.).
	Type string `json:"-"`
	// Default is the statically evaluated fallback value injected when omitted.
	Default any `json:"-"`
	// Enum holds the statically evaluated allowed values list.
	Enum []any `json:"-"`
}

// Handler is a sealed interface guaranteed to be either PipelineHandler or OpenAPIHandler.
type Handler interface {
	isHandler()
}

// PipelineHandler represents an endpoint executed through an ordered sequence of steps.
type PipelineHandler struct {
	Steps []ParsedStep
}

func (PipelineHandler) isHandler() {}

// OpenAPIHandler represents an engine-managed endpoint serving specifications or documentation UIs.
type OpenAPIHandler struct {
	Mode        string
	Format      string
	Renderer    string
	SpecURL     string
	Template    string
	Title       string
	Version     string
	Description string
}

func (OpenAPIHandler) isHandler() {}

// Endpoint represents an HTTP route with ingress rules and a verified terminal handler.
type Endpoint struct {
	MethodAndPath string         `hcl:"name,label"`
	Description   *string        `hcl:"description,optional"`
	Request       *RequestBlock  `hcl:"request,block"`
	Pipeline      *PipelineBlock `hcl:"pipeline,block"`
	OpenAPIs      []OpenAPIBlock `hcl:"openapi,block"`
	DeclaringDir  string         `json:"-"`

	// Method is the normalized uppercase HTTP verb ("GET", "POST", etc.).
	Method string `json:"-"`
	// Path is the route pattern segment ("/api/v1/users/{id}").
	Path string `json:"-"`
	// Handler is the sealed terminal handler (PipelineHandler or OpenAPIHandler).
	Handler Handler `json:"-"`
	// RequestRules holds verified field constraints for ingress validation.
	RequestRules RequestRules `json:"-"`
}

// OpenAPIBlock represents the raw HCL openapi block inside an endpoint.
type OpenAPIBlock struct {
	Mode     string   `hcl:"mode,label"`
	Renderer *string  `hcl:"renderer,optional"`
	Format   *string  `hcl:"format,optional"`
	SpecURL  *string  `hcl:"spec_url,optional"`
	File     *string  `hcl:"file,optional"`
	Inline   *string  `hcl:"inline,optional"`
	Remain   hcl.Body `hcl:",remain"`
}

// PipelineBlock captures the raw HCL body of pipeline steps to preserve definition order.
type PipelineBlock struct {
	Body hcl.Body `hcl:",remain"`
}

// RequestBlock represents the raw HCL request block supporting inline and referenced schemas.
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

// FieldGroup captures nested inline field definitions.
type FieldGroup struct {
	Fields []Field  `hcl:"field,block"`
	Remain hcl.Body `hcl:",remain"`
}

// Decode extracts inline field blocks or schema reference expressions for all request targets.
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

// RequestRules holds pre-compiled field validation constraints for an endpoint.
type RequestRules struct {
	PathFields   []Field
	QueryFields  []Field
	HeaderFields []Field
	BodyFields   []Field
}

// StepType defines the runner category for a pipeline step.
type StepType string

const (
	StepTypeGo       StepType = "go"
	StepTypeStarlark StepType = "starlark"
	StepTypeSQL      StepType = "sql"
	StepTypeRespond  StepType = "respond"
)

// ParsedStep defines an unexecuted step in a pipeline.
type ParsedStep struct {
	Type     StepType
	Name     string
	Go       *GoStep
	Starlark *StarlarkStep
	SQL      *SQLStep
	Respond  *RespondStep
}

// GoStep defines invocation parameters for a native Go callback function.
type GoStep struct {
	Use  string         `hcl:"use,attr"`
	Args hcl.Expression `hcl:"args,optional"`
}

// StarlarkStep defines source code for a sandboxed Starlark script.
type StarlarkStep struct {
	Source string `hcl:"source,attr"`
}

// SQLStep defines a parameterized database query or mutation with catch blocks.
type SQLStep struct {
	Connection hcl.Expression `hcl:"connection,attr"`
	Query      string         `hcl:"query,attr"`
	Args       hcl.Expression `hcl:"args,optional"`
	Catches    []SQLCatch     `hcl:"catch,block"`
}

// SQLCatch defines error-code matching and response mapping for a SQL query failure.
type SQLCatch struct {
	Code    string         `hcl:"code,label"`
	Status  hcl.Expression `hcl:"status,optional"`
	Headers hcl.Expression `hcl:"headers,optional"`
	Body    hcl.Expression `hcl:"body,optional"`
}

// RespondStep defines the condition, headers, status, and payload to write to the client.
type RespondStep struct {
	Condition hcl.Expression `hcl:"condition,optional"`
	Status    hcl.Expression `hcl:"status,optional"`
	Headers   hcl.Expression `hcl:"headers,optional"`
	Body      hcl.Expression `hcl:"body,optional"`
}
