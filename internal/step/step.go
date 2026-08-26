package step

import "github.com/ju4n97/hclapi"

// Runner is the runnable interface for all steps in a pipeline.
type Runner interface {
	Run(ctx *hclapi.Context) error
}
