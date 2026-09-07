package engine

import (
	"fmt"
	"net/http"

	"github.com/ju4n97/hclapi/internal/runtime"
)

// StepResult represents the outcome of executing a pipeline step.
type StepResult struct {
	// Terminated indicates whether the step sent an HTTP response and subsequent steps should be skipped.
	Terminated bool
}

// Step defines the execution contract for an individual pipeline step.
type Step interface {
	Run(execCtx *runtime.ExecutionContext, w http.ResponseWriter) (StepResult, error)
}

// Pipeline coordinates sequential step execution for an endpoint.
type Pipeline struct {
	steps []Step
}

// NewPipeline creates a new pipeline from an ordered slice of Steps.
func NewPipeline(steps ...Step) *Pipeline {
	return &Pipeline{steps: steps}
}

// Execute runs configured steps in order until completion or pipeline termination.
func (p *Pipeline) Execute(execCtx *runtime.ExecutionContext, w http.ResponseWriter) error {
	for _, step := range p.steps {
		if err := execCtx.Context().Err(); err != nil {
			return fmt.Errorf("pipeline execution aborted: %w", err)
		}

		res, err := step.Run(execCtx, w)
		if err != nil {
			return err
		}

		if res.Terminated {
			return nil
		}
	}

	return nil
}
