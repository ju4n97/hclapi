package parser

import "github.com/hashicorp/hcl/v2"

// Manifest is the root structure representing all merged HCL files.
type Manifest struct {
	Endpoints []Endpoint `hcl:"endpoint,block"`
}

// Endpoint represents a single HTTP route definition.
type Endpoint struct {
	MethodAndPath string   `hcl:"name,label"`
	Description   *string  `hcl:"description,attr"`
	Pipeline      Pipeline `hcl:"pipeline,block"`
}

// Pipeline holds the raw HCL body so steps are evaluated in exact source order.
type Pipeline struct {
	Body hcl.Body `hcl:",remain"`
}

// StepType identifies the runner type for a step.
type StepType string

const (
	StepTypeGo       StepType = "go"
	StepTypeStarlark StepType = "starlark"
	StepTypeRespond  StepType = "respond"
)

// ParsedStep is an intermediate representation of an ordered pipeline step.
type ParsedStep struct {
	Type     StepType
	Name     string // e.g. 'hash_token' in: go "hash_token" { ... }
	Go       *GoStepConfig
	Starlark *StarlarkStepConfig
	Respond  *RespondStepConfig
}

// GoStepConfig represents configuration for the `go` step.
type GoStepConfig struct {
	Use string `hcl:"use,attr"`
}

// StarlarkStepConfig represents configuration for the `starlark` step.
type StarlarkStepConfig struct {
	Source string `hcl:"source,attr"`
}

// RespondStepConfig represents configuration for the `respond` step.
type RespondStepConfig struct {
	Status int     `hcl:"status,attr"`
	Body   *string `hcl:"body,attr"`
}
