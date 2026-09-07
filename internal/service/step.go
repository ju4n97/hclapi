package service

import "github.com/hashicorp/hcl/v2"

// StepType defines the execution category for a pipeline step.
type StepType string

// StepType constants for supported step types.
const (
	StepTypeGo       StepType = "go"
	StepTypeStarlark StepType = "starlark"
	StepTypeSQL      StepType = "sql"
	StepTypeRespond  StepType = "respond"
)

// Step represents a configured step ready for execution within a pipeline.
type Step struct {
	Type     StepType
	Name     string
	Go       *GoStep
	Starlark *StarlarkStep
	SQL      *SQLStep
	Respond  *RespondStep
}

// GoStep defines invocation parameters for a native Go callback function.
type GoStep struct {
	Use  string
	Args hcl.Expression
}

// StarlarkStep defines source code for an embedded Starlark transformation.
type StarlarkStep struct {
	Source string
}

// SQLStep defines a parameterized database query with constraint catch blocks.
type SQLStep struct {
	ConnectionKey string
	Query         string
	Args          hcl.Expression
	Catches       []SQLCatch
}

// SQLCatch defines response mapping when a database query raises a specific error code.
type SQLCatch struct {
	Code    string
	Status  hcl.Expression
	Headers hcl.Expression
	Body    hcl.Expression
}

// RespondStep defines the status code, headers, and body to return to the HTTP client.
type RespondStep struct {
	Condition hcl.Expression
	Status    hcl.Expression
	Headers   hcl.Expression
	Body      hcl.Expression
}
