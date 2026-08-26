package parser

import "github.com/hashicorp/hcl/v2"

// Manifest represents the root collection of merged HCL route definitions.
type Manifest struct {
	Server      *Server      `hcl:"server,block"`
	Connections []Connection `hcl:"connection,block"`
	Schemas     []Schema     `hcl:"schema,block"`
	Endpoints   []Endpoint   `hcl:"endpoint,block"`
	Remain      hcl.Body     `hcl:",remain"`
}

// Server represents the global server configuration.
type Server struct {
	Host         *string  `hcl:"host,attr"`
	Port         *int     `hcl:"port,attr"`
	ReadTimeout  *string  `hcl:"read_timeout,attr"`
	WriteTimeout *string  `hcl:"write_timeout,attr"`
	Remain       hcl.Body `hcl:",remain"`
}

// Connection represents a connection pool configuration.
type Connection struct {
	Type   string   `hcl:"type,attr"`
	Name   string   `hcl:"name,label"`
	URL    string   `hcl:"url,attr"`
	Remain hcl.Body `hcl:",remain"`
}

// Schema represents a request body schema definition.
type Schema struct {
	Name   string   `hcl:"name,label"`
	Remain hcl.Body `hcl:",remain"`
}

// Endpoint represents a single HTTP route declaration and its execution pipeline.
type Endpoint struct {
	MethodAndPath string   `hcl:"name,label"`
	Description   *string  `hcl:"description,attr"`
	Request       *Request `hcl:"request,block"`
	Pipeline      Pipeline `hcl:"pipeline,block"`
	Remain        hcl.Body `hcl:",remain"`
}

// Request represents the request body schema and validation.
type Request struct {
	Remain hcl.Body `hcl:",remain"`
}

// Pipeline encapsulates the raw HCL body of pipeline steps to preserve definition order.
type Pipeline struct {
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
	Go       *GoStepConfig
	Starlark *StarlarkStepConfig
	Respond  *RespondStepConfig
}

// GoStepConfig defines invocation settings for a custom Go handler.
type GoStepConfig struct {
	Use  string         `hcl:"use,attr"`
	Args hcl.Expression `hcl:"args,optional"`
}

// StarlarkStepConfig defines the script source for a Starlark step.
type StarlarkStepConfig struct {
	Source string `hcl:"source,attr"`
}

// RespondStepConfig defines the parameters for terminating and serializing an HTTP response.
type RespondStepConfig struct {
	Condition hcl.Expression `hcl:"condition,optional"`
	Status    hcl.Expression `hcl:"status,optional"`
	Body      hcl.Expression `hcl:"body,optional"`
}
