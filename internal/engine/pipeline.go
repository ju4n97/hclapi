package engine

import (
	"errors"
	"fmt"
	"net/http"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/parser"
	"github.com/ju4n97/hclapi/internal/steps/xgo"
	"github.com/ju4n97/hclapi/internal/steps/xrespond"
	"github.com/ju4n97/hclapi/internal/steps/xsql"
	"github.com/ju4n97/hclapi/internal/steps/xstarlark"
)

// PipelineExecutor coordinates sequential pipeline step dispatching.
type PipelineExecutor struct {
	steps      []parser.ParsedStep
	goSteps    map[string]core.StepHandler
	sqlManager *connsql.Manager
}

// NewPipelineExecutor creates a new instance of the pipeline runner.
func NewPipelineExecutor(
	steps []parser.ParsedStep,
	goSteps map[string]core.StepHandler,
	sqlManager *connsql.Manager,
) *PipelineExecutor {
	return &PipelineExecutor{
		steps:      steps,
		goSteps:    goSteps,
		sqlManager: sqlManager,
	}
}

// Execute walks through configured steps sequentially, updating context state.
func (p *PipelineExecutor) Execute(w http.ResponseWriter, ctx *core.Context) error {
	for _, step := range p.steps {
		if err := ctx.Context().Err(); err != nil {
			return fmt.Errorf("pipeline execution aborted: %w", err)
		}

		switch step.Type {
		case parser.StepTypeGo:
			if err := p.execGoStep(step, ctx); err != nil {
				return err
			}

		case parser.StepTypeStarlark:
			if err := p.execStarlarkStep(step, ctx); err != nil {
				return err
			}

		case parser.StepTypeSQL:
			if err := p.execSQLStep(step, ctx); err != nil {
				return err
			}

		case parser.StepTypeRespond:
			responded, err := p.execRespondStep(w, step, ctx)
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

func (p *PipelineExecutor) execGoStep(step parser.ParsedStep, ctx *core.Context) error {
	if step.Go == nil {
		return fmt.Errorf("step %q: missing go configuration", step.Name)
	}

	args, err := eval.Map(step.Go.Args, ctx)
	if err != nil {
		return fmt.Errorf("step %q args: %w", step.Name, err)
	}

	handler, exists := p.goSteps[step.Go.Use]
	if !exists {
		return fmt.Errorf("step %q: unregistered go function %q", step.Name, step.Go.Use)
	}

	res, err := xgo.Execute(handler, ctx, args)
	if err != nil {
		return fmt.Errorf("step %q: %w", step.Name, err)
	}

	if step.Name != "" {
		ctx.Steps[step.Name] = map[string]any{
			"result": res,
		}
	}

	return nil
}

func (p *PipelineExecutor) execStarlarkStep(step parser.ParsedStep, ctx *core.Context) error {
	if step.Starlark == nil {
		return fmt.Errorf("step %q: missing starlark configuration", step.Name)
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
		return fmt.Errorf("step %q: %w", step.Name, err)
	}

	if step.Name != "" {
		ctx.Steps[step.Name] = map[string]any{
			"result": res,
		}
	}

	return nil
}

func (p *PipelineExecutor) execSQLStep(step parser.ParsedStep, ctx *core.Context) error {
	if step.SQL == nil {
		return fmt.Errorf("step %q: missing sql configuration", step.Name)
	}

	connRef, err := parser.ResolveConnectionRef(step.SQL.Connection)
	if err != nil {
		return fmt.Errorf("step %q connection: %w", step.Name, err)
	}

	pool, exists := p.sqlManager.Get(connRef)
	if !exists {
		return fmt.Errorf("step %q: unknown connection %q", step.Name, connRef)
	}

	args, err := eval.Map(step.SQL.Args, ctx)
	if err != nil {
		return fmt.Errorf("step %q args: %w", step.Name, err)
	}

	res, err := xsql.Execute(ctx.Context(), pool, step.SQL.Query, args)
	if err != nil {
		return fmt.Errorf("step %q: %w", step.Name, err)
	}

	if step.Name != "" {
		ctx.Steps[step.Name] = map[string]any{
			"rows":          res.Rows,
			"row":           res.Row,
			"rows_affected": res.RowsAffected,
		}
	}

	return nil
}

func (p *PipelineExecutor) execRespondStep(
	w http.ResponseWriter,
	step parser.ParsedStep,
	ctx *core.Context,
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

	// Evaluate headers if defined
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

	// Evaluate body if defined
	var body any
	if step.Respond.Body != nil {
		evaluatedBody, err := eval.Any(step.Respond.Body, ctx)
		if err != nil {
			return false, fmt.Errorf("respond body evaluation failed: %w", err)
		}
		body = evaluatedBody
	}

	if err := xrespond.Write(w, status, headers, body); err != nil {
		return false, fmt.Errorf("failed to write response: %w", err)
	}

	return true, nil // Terminal response successfully written
}
