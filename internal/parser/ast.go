package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/ju4n97/hclapi/internal/core"
)

// Manifest represents the root collection of merged HCL route definitions.
type Manifest struct {
	Server      *ServerBlock      `hcl:"server,block"`
	Connections []ConnectionBlock `hcl:"connection,block"`
	Schemas     []SchemaBlock     `hcl:"schema,block"`
	Endpoints   []EndpointBlock   `hcl:"endpoint,block"`
	Remain      hcl.Body          `hcl:",remain"`
}

// ServerBlock represents the raw HCL server syntax block.
type ServerBlock struct {
	Host         string   `hcl:"host,optional"`
	Port         int      `hcl:"port,optional"`
	ReadTimeout  *string  `hcl:"read_timeout,optional"`
	WriteTimeout *string  `hcl:"write_timeout,optional"`
	IdleTimeout  *string  `hcl:"idle_timeout,optional"`
	MaxBodySize  *string  `hcl:"max_body_size,optional"`
	ErrorBaseURL *string  `hcl:"error_base_url,optional"`
	Remain       hcl.Body `hcl:",remain"`
}

// ToServer maps the AST ServerBlock into a pure domain core.Server with defaults applied.
func (s *ServerBlock) ToServer() (core.Server, error) {
	def := core.DefaultServer()
	if s == nil {
		return def, nil
	}

	srv := core.Server{
		Host: s.Host,
		Port: s.Port,
	}

	if s.ReadTimeout != nil {
		var d core.Duration
		if err := d.UnmarshalText([]byte(*s.ReadTimeout)); err != nil {
			return core.Server{}, fmt.Errorf("server: invalid read_timeout: %w", err)
		}
		srv.ReadTimeout = d
	}
	if s.WriteTimeout != nil {
		var d core.Duration
		if err := d.UnmarshalText([]byte(*s.WriteTimeout)); err != nil {
			return core.Server{}, fmt.Errorf("server: invalid write_timeout: %w", err)
		}
		srv.WriteTimeout = d
	}
	if s.IdleTimeout != nil {
		var d core.Duration
		if err := d.UnmarshalText([]byte(*s.IdleTimeout)); err != nil {
			return core.Server{}, fmt.Errorf("server: invalid idle_timeout: %w", err)
		}
		srv.IdleTimeout = d
	}
	if s.MaxBodySize != nil {
		b, err := core.ParseByteSize(*s.MaxBodySize)
		if err != nil {
			return core.Server{}, fmt.Errorf("server: invalid max_body_size: %w", err)
		}
		srv.MaxBodySize = b
	}
	if s.ErrorBaseURL != nil {
		srv.ErrorBaseURL = *s.ErrorBaseURL
	}

	return srv.WithDefaults(), nil
}

// ConnectionBlock represents a connection pool configuration block.
type ConnectionBlock struct {
	Driver string               `hcl:"driver,label"`
	Name   string               `hcl:"name,label"`
	URL    string               `hcl:"url,attr"`
	Pool   *ConnectionPoolBlock `hcl:"pool,block"`
	Remain hcl.Body             `hcl:",remain"`
}

// ConnectionPoolBlock represents connection pool tuning parameters.
type ConnectionPoolBlock struct {
	MaxOpenConns    *int     `hcl:"max_open_conns,optional"`
	MaxIdleConns    *int     `hcl:"max_idle_conns,optional"`
	ConnMaxLifetime *string  `hcl:"conn_max_lifetime,optional"`
	IdleTimeout     *string  `hcl:"idle_timeout,optional"`
	Size            *int     `hcl:"size,optional"`
	Remain          hcl.Body `hcl:",remain"`
}

// ToConnection maps the AST ConnectionBlock into a pure domain core.Connection with defaults applied.
func (c *ConnectionBlock) ToConnection() (core.Connection, error) {
	conn := core.Connection{
		Driver: c.Driver,
		Name:   c.Name,
		URL:    c.URL,
		Pool:   core.DefaultPoolConfig(),
	}

	if c.Pool != nil {
		if c.Pool.MaxOpenConns != nil {
			conn.Pool.MaxOpenConns = *c.Pool.MaxOpenConns
		}
		if c.Pool.MaxIdleConns != nil {
			conn.Pool.MaxIdleConns = *c.Pool.MaxIdleConns
		}
		if c.Pool.ConnMaxLifetime != nil {
			var d core.Duration
			if err := d.UnmarshalText([]byte(*c.Pool.ConnMaxLifetime)); err != nil {
				return core.Connection{}, fmt.Errorf("connection %q: invalid conn_max_lifetime: %w", c.Name, err)
			}
			conn.Pool.ConnMaxLifetime = d
		}
		if c.Pool.IdleTimeout != nil {
			var d core.Duration
			if err := d.UnmarshalText([]byte(*c.Pool.IdleTimeout)); err != nil {
				return core.Connection{}, fmt.Errorf("connection %q: invalid idle_timeout: %w", c.Name, err)
			}
			conn.Pool.IdleTimeout = d
		}
		if c.Pool.Size != nil {
			conn.Pool.Size = *c.Pool.Size
		}
	}

	return conn, nil
}

// FieldBlock defines a single field's data type, presence, defaults, and validation constraints.
type FieldBlock struct {
	Name        string         `hcl:"name,label"`
	Type        hcl.Expression `hcl:"type,attr"`
	Required    bool           `hcl:"required,optional"`
	Default     hcl.Expression `hcl:"default,optional"`
	Description *string        `hcl:"description,optional"`
	Enum        hcl.Expression `hcl:"enum,optional"`
	Format      *string        `hcl:"format,optional"`
	Pattern     *string        `hcl:"pattern,optional"`
	MinLength   *int           `hcl:"min_length,optional"`
	MaxLength   *int           `hcl:"max_length,optional"`
	Min         *float64       `hcl:"min,optional"`
	Max         *float64       `hcl:"max,optional"`
	MinItems    *int           `hcl:"min_items,optional"`
	MaxItems    *int           `hcl:"max_items,optional"`
	UniqueItems bool           `hcl:"unique_items,optional"`
	Remain      hcl.Body       `hcl:",remain"`
}

// FieldGroupBlock represents a collection of field validation rules (for path, query, headers, or inline body).
type FieldGroupBlock struct {
	Fields []FieldBlock `hcl:"field,block"`
	Remain hcl.Body     `hcl:",remain"`
}

// SchemaBlock represents a reusable request payload schema definition block.
type SchemaBlock struct {
	Name        string       `hcl:"name,label"`
	Description *string      `hcl:"description,optional"`
	Fields      []FieldBlock `hcl:"field,block"`
	Remain      hcl.Body     `hcl:",remain"`
}

// RequestBlock represents parameter and body validation rules for an endpoint.
type RequestBlock struct {
	Path       *FieldGroupBlock `hcl:"path,block"`
	Query      *FieldGroupBlock `hcl:"query,block"`
	Headers    *FieldGroupBlock `hcl:"headers,block"`
	Remain     hcl.Body         `hcl:",remain"`
	BodyExpr   hcl.Expression   // Populated if `body = schema.name` attribute is used
	BodyInline *FieldGroupBlock // Populated if inline `body { field ... }` block is used
}

// DecodeBody decodes the body attribute expression or inline body block from the request body remainder.
func (r *RequestBlock) DecodeBody(ctx *hcl.EvalContext) error {
	if r == nil || r.Remain == nil {
		return nil
	}

	content, _, diags := r.Remain.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "body", Required: false},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "body"},
		},
	})
	if diags.HasErrors() {
		return fmt.Errorf("request body: %w", diags)
	}

	if attr, ok := content.Attributes["body"]; ok {
		r.BodyExpr = attr.Expr
	}

	for _, block := range content.Blocks {
		if block.Type == "body" {
			var inline FieldGroupBlock
			if err := gohcl.DecodeBody(block.Body, ctx, &inline); err.HasErrors() {
				return fmt.Errorf("inline body block: %w", err)
			}
			r.BodyInline = &inline
		}
	}

	return nil
}

// EndpointBlock represents a single HTTP route declaration block.
type EndpointBlock struct {
	MethodAndPath string        `hcl:"name,label"`
	Description   *string       `hcl:"description,attr"`
	Request       *RequestBlock `hcl:"request,block"`
	Pipeline      PipelineBlock `hcl:"pipeline,block"`
	Remain        hcl.Body      `hcl:",remain"`
}

// PipelineBlock encapsulates the raw HCL body of pipeline steps to preserve definition order.
type PipelineBlock struct {
	Body hcl.Body `hcl:",remain"`
}

// StepType defines the runner category for a pipeline step.
type StepType string

const (
	StepTypeGo       StepType = "go"
	StepTypeStarlark StepType = "starlark"
	StepTypeSQL      StepType = "sql"
	StepTypeRespond  StepType = "respond"
)

// ParsedStep is an intermediate representation of a sequential step in a pipeline.
type ParsedStep struct {
	Type     StepType
	Name     string
	Go       *GoStepBlock
	Starlark *StarlarkStepBlock
	SQL      *SQLStepBlock
	Respond  *RespondStepBlock
}

// GoStepBlock defines invocation settings for a custom Go handler step.
type GoStepBlock struct {
	Use  string         `hcl:"use,attr"`
	Args hcl.Expression `hcl:"args,optional"`
}

// StarlarkStepBlock defines the script source for a Starlark step.
type StarlarkStepBlock struct {
	Source string `hcl:"source,attr"`
}

// SQLStepBlock defines the SQL query to execute.
type SQLStepBlock struct {
	Connection hcl.Expression  `hcl:"connection,attr"`
	Query      string          `hcl:"query,attr"`
	Args       hcl.Expression  `hcl:"args,optional"`
	Catches    []SQLCatchBlock `hcl:"catch,block"`
	Remain     hcl.Body        `hcl:",remain"`
}

// SQLCatchBlock defines error code interception and response payload mapping for a SQL step.
type SQLCatchBlock struct {
	Code    string         `hcl:"code,label"`
	Status  hcl.Expression `hcl:"status,optional"`
	Headers hcl.Expression `hcl:"headers,optional"`
	Body    hcl.Expression `hcl:"body,optional"`
	Remain  hcl.Body       `hcl:",remain"`
}

// RespondStepBlock defines the parameters for serializing an HTTP response.
type RespondStepBlock struct {
	Condition hcl.Expression `hcl:"condition,optional"`
	Status    hcl.Expression `hcl:"status,optional"`
	Headers   hcl.Expression `hcl:"headers,optional"`
	Body      hcl.Expression `hcl:"body,optional"`
}

// ResolveConnectionRef extracts the connection identifier string from an HCL expression.
// It handles unquoted traversals (connection.postgres.main) and string literals (connection: "postgres.main").
func ResolveConnectionRef(expr hcl.Expression) (string, error) {
	if expr == nil {
		return "", errors.New("missing connection reference")
	}

	// If it's a traversal
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

	// If it's a string literal or evaluated expression
	val, diags := expr.Value(nil)
	if !diags.HasErrors() && val.Type().Equals(cty.String) {
		return val.AsString(), nil
	}

	return "", errors.New("invalid connection reference expression")
}
