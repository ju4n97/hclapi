package config

import (
	"github.com/hashicorp/hcl/v2"
)

// Handler is a sealed interface guaranteed to be either PipelineHandler or OpenAPIHandler.
type Handler interface {
	isHandler()
}

// PipelineHandler executes a sequence of steps.
type PipelineHandler struct {
	Steps []ParsedStep
}

func (PipelineHandler) isHandler() {}

// OpenAPIHandler renders an OpenAPI 3.1 document.
type OpenAPIHandler struct {
	Mode        string // "spec", "ui", "template"
	Format      string // "json", "yaml"
	Renderer    string // "scalar", "elements", "swagger", "redoc"
	SpecURL     string
	Template    string // Loaded template content (from inline heredoc or external file)
	Title       string
	Version     string
	Description string
}

func (OpenAPIHandler) isHandler() {}

// Endpoint represents a single HTTP route with ingress rules and a verified handler.
type Endpoint struct {
	MethodAndPath string
	Method        string
	Path          string
	Description   string
	RequestRules  RequestRules
	Handler       Handler
}

// RequestRules holds verified field constraints for request ingress.
type RequestRules struct {
	PathFields   []Field
	QueryFields  []Field
	HeaderFields []Field
	BodyFields   []Field
}

// StepType defines the type of a parsed pipeline step.
type StepType string

// StepType constants for supported step types.
const (
	StepTypeGo       StepType = "go"
	StepTypeStarlark StepType = "starlark"
	StepTypeSQL      StepType = "sql"
	StepTypeRespond  StepType = "respond"
)

// ParsedStep is a parsed step definition from the HCL configuration.
type ParsedStep struct {
	Type     StepType
	Name     string
	Go       *GoStep
	Starlark *StarlarkStep
	SQL      *SQLStep
	Respond  *RespondStep
}

// GoStep executes a registered native Go function.
type GoStep struct {
	Use  string         `hcl:"use,attr"`
	Args hcl.Expression `hcl:"args,optional"`
}

// StarlarkStep executes a sandboxed Starlark script.
type StarlarkStep struct {
	Source string `hcl:"source,attr"`
}

// SQLStep executes a database query or mutation with constraint catch blocks.
type SQLStep struct {
	Connection hcl.Expression `hcl:"connection,attr"`
	Query      string         `hcl:"query,attr"`
	Args       hcl.Expression `hcl:"args,optional"`
	Catches    []SQLCatch     `hcl:"catch,block"`
}

// SQLCatch defines a constraint catch block for SQLStep.
type SQLCatch struct {
	Code    string         `hcl:"code,label"`
	Status  hcl.Expression `hcl:"status,optional"`
	Headers hcl.Expression `hcl:"headers,optional"`
	Body    hcl.Expression `hcl:"body,optional"`
}

// RespondStep serializes HTTP headers, status code, and payload.
type RespondStep struct {
	Condition hcl.Expression `hcl:"condition,optional"`
	Status    hcl.Expression `hcl:"status,optional"`
	Headers   hcl.Expression `hcl:"headers,optional"`
	Body      hcl.Expression `hcl:"body,optional"`
}
