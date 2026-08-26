package engine

import (
	"fmt"
	"net/http"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
	"github.com/ju4n97/hclapi/internal/steps/xgo"
	"github.com/ju4n97/hclapi/internal/steps/xrespond"
	"github.com/ju4n97/hclapi/internal/steps/xstarlark"
)

// PipelineExecutor handles sequential step dispatching.
type PipelineExecutor struct {
	steps   []parser.ParsedStep
	goSteps map[string]core.StepHandler
}

// NewPipelineExecutor creates a new instance of the sequential pipeline runner.
func NewPipelineExecutor(steps []parser.ParsedStep, goSteps map[string]core.StepHandler) *PipelineExecutor {
	return &PipelineExecutor{
		steps:   steps,
		goSteps: goSteps,
	}
}

// Execute walks through configured steps sequentially, updating context state.
func (p *PipelineExecutor) Execute(w http.ResponseWriter, ctx *core.Context) error {
	var lastResult any

	for _, step := range p.steps {
		switch step.Type {
		case parser.StepTypeGo:
			if step.Go == nil {
				return fmt.Errorf("step %q is missing go configuration", step.Name)
			}

			args, err := eval.EvalMap(&step.Go.Args, ctx)
			if err != nil {
				return fmt.Errorf("step %q args evaluation failed: %w", step.Name, err)
			}
			ctx.Args = args

			handler, exists := p.goSteps[step.Go.Use]
			if !exists {
				return fmt.Errorf("unregistered go function %q", step.Go.Use)
			}

			res, err := xgo.Execute(handler, ctx)
			if err != nil {
				return fmt.Errorf("step %q execution failed: %w", step.Name, err)
			}

			if step.Name != "" {
				ctx.Steps[step.Name] = core.StepResult{Result: res}
			}
			lastResult = res

		case parser.StepTypeStarlark:
			if step.Starlark == nil {
				return fmt.Errorf("step %q is missing starlark configuration", step.Name)
			}

			ctxData := map[string]any{
				"request": map[string]any{
					"method":  ctx.Request.Method,
					"path":    ctx.Request.Path,
					"query":   ctx.Request.Query,
					"headers": ctx.Request.Headers,
					"body":    ctx.Request.Body,
				},
				"steps":           map[string]any{},
				"timestamp_epoch": ctx.TimestampEpoch,
			}
			for k, v := range ctx.Steps {
				ctxData["steps"].(map[string]any)[k] = v.Result
			}

			res, err := xstarlark.Eval(step.Starlark.Source, ctxData)
			if err != nil {
				return fmt.Errorf("step %q starlark execution failed: %w", step.Name, err)
			}

			if step.Name != "" {
				ctx.Steps[step.Name] = core.StepResult{Result: res}
			}
			lastResult = res

		case parser.StepTypeRespond:
			if step.Respond == nil {
				return fmt.Errorf("step is missing respond configuration")
			}

			shouldRun, err := eval.EvalBool(&step.Respond.Condition, ctx, true)
			if err != nil {
				return fmt.Errorf("respond condition evaluation failed: %w", err)
			}
			if !shouldRun {
				continue // Condition evaluated to false; evaluate next step
			}

			status, err := eval.EvalInt(&step.Respond.Status, ctx, http.StatusOK)
			if err != nil {
				return fmt.Errorf("respond status evaluation failed: %w", err)
			}

			body, err := eval.EvalAny(&step.Respond.Body, ctx)
			if err != nil {
				return fmt.Errorf("respond body evaluation failed: %w", err)
			}

			return xrespond.Write(w, status, body, lastResult)
		}
	}

	return nil
}
