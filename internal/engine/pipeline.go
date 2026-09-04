package engine

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/hcl/v2"
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
func (p *PipelineExecutor) Execute(w http.ResponseWriter, execCtx *core.ExecutionContext) error {
	for _, step := range p.steps {
		if err := execCtx.Context().Err(); err != nil {
			return fmt.Errorf("pipeline execution aborted: %w", err)
		}

		switch step.Type {
		case parser.StepTypeGo:
			if err := p.execGoStep(step, execCtx); err != nil {
				return err
			}

		case parser.StepTypeStarlark:
			if err := p.execStarlarkStep(step, execCtx); err != nil {
				return err
			}

		case parser.StepTypeSQL:
			responded, err := p.execSQLStep(w, step, execCtx)
			if err != nil {
				return err
			}
			if responded {
				return nil // Terminal catch step reached and written
			}

		case parser.StepTypeRespond:
			responded, err := p.execRespondStep(w, step, execCtx)
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

func (p *PipelineExecutor) execGoStep(step parser.ParsedStep, execCtx *core.ExecutionContext) error {
	if step.Go == nil {
		return fmt.Errorf("step %q: missing go configuration", step.Name)
	}

	argsMap, err := eval.Map(step.Go.Args, execCtx)
	if err != nil {
		return fmt.Errorf("step %q args: %w", step.Name, err)
	}

	handler, exists := p.goSteps[step.Go.Use]
	if !exists {
		return fmt.Errorf("step %q: unregistered go function %q", step.Name, step.Go.Use)
	}

	stepObj := execCtx.NewStep(step.Name, core.Args(argsMap))

	res, err := xgo.Execute(execCtx.Context(), handler, stepObj)
	if err != nil {
		return fmt.Errorf("step %q: %w", step.Name, err)
	}

	if step.Name != "" {
		execCtx.SetStepResult(step.Name, map[string]any{
			"result": res,
		})
	}

	return nil
}

func (p *PipelineExecutor) execStarlarkStep(step parser.ParsedStep, execCtx *core.ExecutionContext) error {
	if step.Starlark == nil {
		return fmt.Errorf("step %q: missing starlark configuration", step.Name)
	}

	reqFields := starlark.StringDict{
		"method":  starlark.String(execCtx.Request.Method),
		"path":    xstarlark.GoToStarlarkValue(execCtx.Request.Path),
		"query":   xstarlark.GoToStarlarkValue(execCtx.Request.Query),
		"headers": xstarlark.NewCaseInsensitiveDictFromStrings(execCtx.Request.Headers),
		"body":    xstarlark.GoToStarlarkValue(execCtx.Request.Body),
	}

	steps := execCtx.SnapshotSteps()
	stepFields := make(starlark.StringDict, len(steps))
	for name, stepExports := range steps {
		stepFields[name] = xstarlark.GoToStarlarkValue(stepExports)
	}

	starlarkCtx := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"request":         starlarkstruct.FromStringDict(starlarkstruct.Default, reqFields),
		"steps":           starlarkstruct.FromStringDict(starlarkstruct.Default, stepFields),
		"timestamp_epoch": starlark.MakeInt64(execCtx.TimestampEpoch),
	})

	res, err := xstarlark.Eval(step.Starlark.Source, starlarkCtx)
	if err != nil {
		return fmt.Errorf("step %q: %w", step.Name, err)
	}

	if step.Name != "" {
		execCtx.SetStepResult(step.Name, map[string]any{
			"result": res,
		})
	}

	return nil
}

func (p *PipelineExecutor) execSQLStep(
	w http.ResponseWriter,
	step parser.ParsedStep,
	execCtx *core.ExecutionContext,
) (bool, error) {
	if step.SQL == nil {
		return false, fmt.Errorf("step %q: missing sql configuration", step.Name)
	}

	connRef, err := parser.ResolveConnectionRef(step.SQL.Connection)
	if err != nil {
		return false, fmt.Errorf("step %q connection: %w", step.Name, err)
	}

	pool, exists := p.sqlManager.Get(connRef)
	if !exists {
		return false, fmt.Errorf("step %q: unknown connection %q", step.Name, connRef)
	}

	args, err := eval.Map(step.SQL.Args, execCtx)
	if err != nil {
		return false, fmt.Errorf("step %q args: %w", step.Name, err)
	}

	res, err := xsql.Execute(execCtx.Context(), pool, step.SQL.Query, args)
	if err != nil {
		// Attempt to intercept database constraint error codes with catch blocks
		errCode := pool.Dialect.ExtractErrorCode(err)
		if errCode != "" && len(step.SQL.Catches) > 0 {
			for _, catchBlock := range step.SQL.Catches {
				if pool.Dialect.MatchErrorCode(errCode, catchBlock.Code) {
					if err := writeResponse(
						w,
						execCtx,
						catchBlock.Status,
						catchBlock.Headers,
						catchBlock.Body,
						http.StatusBadRequest,
					); err != nil {
						return false, fmt.Errorf("step %q catch: %w", step.Name, err)
					}
					return true, nil
				}
			}
		}

		return false, fmt.Errorf("step %q: %w", step.Name, err)
	}

	if step.Name != "" {
		execCtx.SetStepResult(step.Name, map[string]any{
			"rows":          res.Rows,
			"row":           res.Row,
			"rows_affected": res.RowsAffected,
		})
	}

	return false, nil
}

func (p *PipelineExecutor) execRespondStep(
	w http.ResponseWriter,
	step parser.ParsedStep,
	execCtx *core.ExecutionContext,
) (bool, error) {
	if step.Respond == nil {
		return false, errors.New("step is missing respond configuration")
	}

	shouldRun, err := eval.Bool(step.Respond.Condition, execCtx, true)
	if err != nil {
		return false, fmt.Errorf("respond condition: %w", err)
	}
	if !shouldRun {
		return false, nil
	}

	if err := writeResponse(w, execCtx, step.Respond.Status, step.Respond.Headers, step.Respond.Body, http.StatusOK); err != nil {
		return false, fmt.Errorf("step %q: %w", step.Name, err)
	}

	return true, nil // Terminal response successfully written
}

// writeResponse evaluates dynamic status, headers, and body expressions and writes the finalized HTTP response.
func writeResponse(
	w http.ResponseWriter,
	execCtx *core.ExecutionContext,
	statusExpr, headersExpr, bodyExpr hcl.Expression,
	defaultStatus int,
) error {
	status, err := eval.Int(statusExpr, execCtx, defaultStatus)
	if err != nil {
		return fmt.Errorf("eval status: %w", err)
	}

	var headers map[string]string
	if headersExpr != nil {
		evaluatedHeaders, err := eval.Map(headersExpr, execCtx)
		if err != nil {
			return fmt.Errorf("eval headers: %w", err)
		}
		if len(evaluatedHeaders) > 0 {
			headers = make(map[string]string, len(evaluatedHeaders))
			for k, v := range evaluatedHeaders {
				headers[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	var body any
	if bodyExpr != nil {
		evaluatedBody, err := eval.Any(bodyExpr, execCtx)
		if err != nil {
			return fmt.Errorf("eval body: %w", err)
		}
		body = evaluatedBody
	}

	if err := xrespond.Write(w, status, headers, body); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
