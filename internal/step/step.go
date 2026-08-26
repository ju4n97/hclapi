package step

import "github.com/ekisa-team/hclapi"

// Runner is the runnable interface for all steps in a pipeline.
type Runner interface {
	Run(ctx *hclapi.Context) error
}
