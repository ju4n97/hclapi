package parser

import (
	"github.com/hashicorp/hcl/v2"

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
func (s *ServerBlock) ToServer() core.Server {
	def := core.DefaultServer()
	if s == nil {
		return def
	}

	srv := core.Server{
		Host: s.Host,
		Port: s.Port,
	}

	if s.ReadTimeout != nil {
		var d core.Duration
		if err := d.UnmarshalText([]byte(*s.ReadTimeout)); err == nil {
			srv.ReadTimeout = d
		}
	}
	if s.WriteTimeout != nil {
		var d core.Duration
		if err := d.UnmarshalText([]byte(*s.WriteTimeout)); err == nil {
			srv.WriteTimeout = d
		}
	}
	if s.IdleTimeout != nil {
		var d core.Duration
		if err := d.UnmarshalText([]byte(*s.IdleTimeout)); err == nil {
			srv.IdleTimeout = d
		}
	}
	if s.MaxBodySize != nil {
		if b, err := core.ParseByteSize(*s.MaxBodySize); err == nil {
			srv.MaxBodySize = b
		}
	}
	if s.ErrorBaseURL != nil {
		srv.ErrorBaseURL = *s.ErrorBaseURL
	}

	return srv.WithDefaults()
}

// ConnectionBlock represents a connection pool configuration block.
type ConnectionBlock struct {
	Type   string   `hcl:"type,attr"`
	Name   string   `hcl:"name,label"`
	URL    string   `hcl:"url,attr"`
	Remain hcl.Body `hcl:",remain"`
}

// SchemaBlock represents a request body schema definition block.
type SchemaBlock struct {
	Name   string   `hcl:"name,label"`
	Remain hcl.Body `hcl:",remain"`
}

// EndpointBlock represents a single HTTP route declaration block.
type EndpointBlock struct {
	MethodAndPath string        `hcl:"name,label"`
	Description   *string       `hcl:"description,attr"`
	Request       *RequestBlock `hcl:"request,block"`
	Pipeline      PipelineBlock `hcl:"pipeline,block"`
	Remain        hcl.Body      `hcl:",remain"`
}

// RequestBlock represents the request validation block.
type RequestBlock struct {
	Remain hcl.Body `hcl:",remain"`
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
	StepTypeRespond  StepType = "respond"
)

// ParsedStep is an intermediate representation of a sequential step in a pipeline.
type ParsedStep struct {
	Type     StepType
	Name     string
	Go       *GoStepBlock
	Starlark *StarlarkStepBlock
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

// RespondStepBlock defines the parameters for serializing an HTTP response.
type RespondStepBlock struct {
	Condition hcl.Expression `hcl:"condition,optional"`
	Status    hcl.Expression `hcl:"status,optional"`
	Headers   hcl.Expression `hcl:"headers,optional"`
	Body      hcl.Expression `hcl:"body,optional"`
}
