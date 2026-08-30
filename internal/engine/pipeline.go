package engine

import (
	"errors"
	"fmt"
	"net/http"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
	"github.com/ju4n97/hclapi/internal/steps/xgo"
	"github.com/ju4n97/hclapi/internal/steps/xrespond"
	"github.com/ju4n97/hclapi/internal/steps/xstarlark"
)

// PipelineExecutor coordinates sequential pipeline step dispatching.
type PipelineExecutor struct {
	steps   []parser.ParsedStep
	goSteps map[string]core.StepHandler
}

// NewPipelineExecutor creates a new instance of the pipeline runner.
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
		if err := ctx.Context().Err(); err != nil {
			return fmt.Errorf("pipeline execution aborted: %w", err)
		}

		switch step.Type {
		case parser.StepTypeGo:
			res, err := p.execGoStep(step, ctx)
			if err != nil {
				return err
			}
			lastResult = res

		case parser.StepTypeStarlark:
			res, err := p.execStarlarkStep(step, ctx)
			if err != nil {
				return err
			}
			lastResult = res

		case parser.StepTypeRespond:
			responded, err := p.execRespondStep(w, step, ctx, lastResult)
			if err != nil {
				return err
			}
			if responded {
				return nil // Terminal step reached and written
			}

		default:
			return fmt.Errorf("unsupported step type %q", step.Type)
		}
	}

	return nil
}

func (p *PipelineExecutor) execGoStep(step parser.ParsedStep, ctx *core.Context) (any, error) {
	if step.Go == nil {
		return nil, fmt.Errorf("step %q is missing go configuration", step.Name)
	}

	args, err := eval.Map(step.Go.Args, ctx)
	if err != nil {
		return nil, fmt.Errorf("step %q args evaluation failed: %w", step.Name, err)
	}
	ctx.Args = args

	handler, exists := p.goSteps[step.Go.Use]
	if !exists {
		return nil, fmt.Errorf("unregistered go function %q", step.Go.Use)
	}

	res, err := xgo.Execute(handler, ctx)
	if err != nil {
		return nil, fmt.Errorf("step %q execution failed: %w", step.Name, err)
	}

	if step.Name != "" {
		ctx.Steps[step.Name] = map[string]any{"result": res}
	}

	return res, nil
}

func (p *PipelineExecutor) execStarlarkStep(step parser.ParsedStep, ctx *core.Context) (any, error) {
	if step.Starlark == nil {
		return nil, fmt.Errorf("step %q is missing starlark configuration", step.Name)
	}

	reqFields := starlark.StringDict{
		"method":  starlark.String(ctx.Request.Method),
		"path":    xstarlark.GoToStarlarkValue(ctx.Request.Path),
		"query":   xstarlark.GoToStarlarkValue(ctx.Request.Query),
		"headers": xstarlark.GoToStarlarkValue(ctx.Request.Headers),
		"body":    xstarlark.GoToStarlarkValue(ctx.Request.Body),
	}

	stepFields := make(starlark.StringDict, len(ctx.Steps))
	for name, stepExports := range ctx.Steps {
		stepFields[name] = xstarlark.GoToStarlarkValue(stepExports)
	}

	starlarkCtx := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"request":         starlarkstruct.FromStringDict(starlarkstruct.Default, reqFields),
		"steps":           starlarkstruct.FromStringDict(starlarkstruct.Default, stepFields),
		"timestamp_epoch": starlark.MakeInt64(ctx.TimestampEpoch),
	})

	res, err := xstarlark.Eval(step.Starlark.Source, starlarkCtx)
	if err != nil {
		return nil, fmt.Errorf("step %q starlark execution failed: %w", step.Name, err)
	}

	if step.Name != "" {
		ctx.Steps[step.Name] = core.StepResult{"result": res}
	}

	return res, nil
}

func (p *PipelineExecutor) execRespondStep(
	w http.ResponseWriter,
	step parser.ParsedStep,
	ctx *core.Context,
	lastResult any,
) (bool, error) {
	if step.Respond == nil {
		return false, errors.New("step is missing respond configuration")
	}

	shouldRun, err := eval.Bool(step.Respond.Condition, ctx, true)
	if err != nil {
		return false, fmt.Errorf("respond condition evaluation failed: %w", err)
	}
	if !shouldRun {
		return false, nil // Condition was false; skip responding and continue pipeline
	}

	status, err := eval.Int(step.Respond.Status, ctx, http.StatusOK)
	if err != nil {
		return false, fmt.Errorf("respond status evaluation failed: %w", err)
	}

	var headers map[string]string
	if step.Respond.Headers != nil {
		evaluatedHeaders, err := eval.Map(step.Respond.Headers, ctx)
		if err != nil {
			return false, fmt.Errorf("respond headers evaluation failed: %w", err)
		}
		if len(evaluatedHeaders) > 0 {
			headers = make(map[string]string, len(evaluatedHeaders))
			for k, v := range evaluatedHeaders {
				headers[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	var body any
	if step.Respond.Body != nil {
		evaluatedBody, err := eval.Any(step.Respond.Body, ctx)
		if err != nil {
			return false, fmt.Errorf("respond body evaluation failed: %w", err)
		}
		body = evaluatedBody
	} else {
		body = lastResult
	}

	if err := xrespond.Write(w, status, headers, body); err != nil {
		return false, fmt.Errorf("failed to write response: %w", err)
	}

	return true, nil // Terminal response successfully written
}
