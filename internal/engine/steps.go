package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/ju4n97/hclapi/internal/config"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/runtime"
	"github.com/ju4n97/hclapi/internal/sqldb"
	"github.com/ju4n97/hclapi/internal/star"
)

// GoStep executes a registered native Go callback function.
type GoStep struct {
	Name     string
	Use      string
	Args     hcl.Expression
	Registry *StepRegistry
}

func (s *GoStep) Run(execCtx *runtime.ExecutionContext, w http.ResponseWriter) (StepResult, error) {
	handler, ok := s.Registry.Get(s.Use)
	if !ok {
		return StepResult{}, fmt.Errorf("step %q: unregistered go function %q", s.Name, s.Use)
	}

	argsMap, err := eval.Map(s.Args, execCtx)
	if err != nil {
		return StepResult{}, fmt.Errorf("step %q args: %w", s.Name, err)
	}

	stepObj := execCtx.NewStep(s.Name, runtime.Args(argsMap))

	res, err := s.executeSafely(execCtx, handler, stepObj)
	if err != nil {
		return StepResult{}, fmt.Errorf("step %q: %w", s.Name, err)
	}

	if s.Name != "" {
		execCtx.SetStepResult(s.Name, map[string]any{"result": res})
	}

	return StepResult{Terminated: false}, nil
}

func (s *GoStep) executeSafely(
	execCtx *runtime.ExecutionContext,
	handler runtime.StepHandler,
	stepObj *runtime.Step,
) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in custom go step %q: %v", s.Name, r)
		}
	}()

	return handler(execCtx.Context(), stepObj)
}

// SQLStep executes a database query or mutation with constraint catch blocks.
type SQLStep struct {
	Name    string
	Pool    *sqldb.Pool
	Query   string
	Args    hcl.Expression
	Catches []config.SQLCatch
}

func (s *SQLStep) Run(execCtx *runtime.ExecutionContext, w http.ResponseWriter) (StepResult, error) {
	args, err := eval.Map(s.Args, execCtx)
	if err != nil {
		return StepResult{}, fmt.Errorf("step %q args: %w", s.Name, err)
	}

	res, err := s.Pool.Execute(execCtx.Context(), s.Query, args)
	if err != nil {
		errCode := s.Pool.Dialect.ExtractErrorCode(err)
		if errCode != "" && len(s.Catches) > 0 {
			for _, catchBlock := range s.Catches {
				if s.Pool.Dialect.MatchErrorCode(errCode, catchBlock.Code) {
					if err := writeResponse(
						w,
						execCtx,
						catchBlock.Status,
						catchBlock.Headers,
						catchBlock.Body,
						http.StatusBadRequest,
					); err != nil {
						return StepResult{}, fmt.Errorf("step %q catch: %w", s.Name, err)
					}
					return StepResult{Terminated: true}, nil
				}
			}
		}

		return StepResult{}, fmt.Errorf("step %q: %w", s.Name, err)
	}

	if s.Name != "" {
		execCtx.SetStepResult(s.Name, map[string]any{
			"rows":          res.Rows,
			"row":           res.Row,
			"rows_affected": res.RowsAffected,
		})
	}

	return StepResult{Terminated: false}, nil
}

// StarlarkStep executes a sandboxed Starlark script.
type StarlarkStep struct {
	Name   string
	Source string
}

func (s *StarlarkStep) Run(execCtx *runtime.ExecutionContext, w http.ResponseWriter) (StepResult, error) {
	env := star.Env{
		Request: star.Request{
			Method:  execCtx.Request.Method,
			Path:    execCtx.Request.Path,
			Query:   execCtx.Request.Query,
			Headers: execCtx.Request.Headers,
			Body:    execCtx.Request.Body,
		},
		Steps:          execCtx.SnapshotSteps(),
		TimestampEpoch: execCtx.TimestampEpoch,
	}

	res, err := star.Eval(s.Source, env)
	if err != nil {
		return StepResult{}, fmt.Errorf("step %q: %w", s.Name, err)
	}

	if s.Name != "" {
		execCtx.SetStepResult(s.Name, map[string]any{"result": res})
	}

	return StepResult{Terminated: false}, nil
}

// RespondStep serializes HTTP headers, status code, and payload.
type RespondStep struct {
	Condition hcl.Expression
	Status    hcl.Expression
	Headers   hcl.Expression
	Body      hcl.Expression
}

func (s *RespondStep) Run(execCtx *runtime.ExecutionContext, w http.ResponseWriter) (StepResult, error) {
	shouldRun, err := eval.Bool(s.Condition, execCtx, true)
	if err != nil {
		return StepResult{}, fmt.Errorf("respond condition: %w", err)
	}
	if !shouldRun {
		return StepResult{Terminated: false}, nil
	}

	if err := writeResponse(w, execCtx, s.Status, s.Headers, s.Body, http.StatusOK); err != nil {
		return StepResult{}, fmt.Errorf("respond: %w", err)
	}

	return StepResult{Terminated: true}, nil
}

func writeResponse(
	w http.ResponseWriter,
	execCtx *runtime.ExecutionContext,
	statusExpr, headersExpr, bodyExpr hcl.Expression,
	defaultStatus int,
) error {
	status, err := eval.Int(statusExpr, execCtx, defaultStatus)
	if err != nil {
		return fmt.Errorf("eval status: %w", err)
	}

	hasContentType := false
	if headersExpr != nil {
		evaluatedHeaders, err := eval.Map(headersExpr, execCtx)
		if err != nil {
			return fmt.Errorf("eval headers: %w", err)
		}
		for k, v := range evaluatedHeaders {
			cleanKey := strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(k))
			cleanVal := strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(fmt.Sprintf("%v", v)))
			if cleanKey == "" {
				continue
			}
			if strings.EqualFold(cleanKey, "Content-Type") {
				hasContentType = true
			}
			w.Header().Set(cleanKey, cleanVal)
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

	if !hasContentType && body != nil && status != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(status)

	if body == nil || status == http.StatusNoContent {
		return nil
	}

	contentType := strings.ToLower(w.Header().Get("Content-Type"))
	if contentType == "" || strings.Contains(contentType, "application/json") || strings.Contains(contentType, "+json") {
		return json.NewEncoder(w).Encode(body)
	}

	switch b := body.(type) {
	case string:
		_, err := w.Write([]byte(b))
		return err
	case []byte:
		_, err := w.Write(b)
		return err
	default:
		_, err := fmt.Fprint(w, b)
		return err
	}
}
