package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/engine"
	"github.com/ju4n97/hclapi/internal/parser"
)

func parseExpr(t *testing.T, src string) hcl.Expression {
	t.Helper()

	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("syntax error in expression %q: %s", src, diags.Error())
	}

	return expr
}

func setupTestSQLiteManager(t *testing.T) *connsql.Manager {
	t.Helper()

	mgr := connsql.NewManager()
	conn := core.Connection{
		Driver: "sqlite",
		Name:   "main",
		URL:    "file:pipeline_test_mem?mode=memory&cache=shared",
		Pool:   core.DefaultPoolConfig(),
	}
	if err := mgr.Open(t.Context(), conn); err != nil {
		t.Fatalf("failed to open test sqlite pool: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	return mgr
}

func TestPipeline_Go(t *testing.T) {
	t.Parallel()

	t.Run("Evaluates args, executes handler, and exports result", func(t *testing.T) {
		t.Parallel()

		goSteps := map[string]core.StepHandler{
			"auth.verify": func(ctx context.Context, step *core.Step) (any, error) {
				token := step.Args.GetOr("token", "")
				return map[string]any{
					"valid": token == "secret-token",
					"uid":   42,
				}, nil
			},
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "auth",
				Go: &parser.GoStepBlock{
					Use:  "auth.verify",
					Args: parseExpr(t, `{ token = ctx.request.headers.authorization }`),
				},
			},
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Status: parseExpr(t, `200`),
					Body:   parseExpr(t, `steps.auth.result`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, goSteps, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		req.Header.Set("Authorization", "secret-token")

		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, execCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["valid"] != true || body["uid"] != float64(42) {
			t.Errorf("unexpected body: %+v", body)
		}
	})

	t.Run("Recovers from panic in custom Go step safely", func(t *testing.T) {
		t.Parallel()

		goSteps := map[string]core.StepHandler{
			"panic.step": func(ctx context.Context, step *core.Step) (any, error) {
				panic("nil pointer dereference inside user step")
			},
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "broken",
				Go:   &parser.GoStepBlock{Use: "panic.step"},
			},
		}

		executor := engine.NewPipelineExecutor(steps, goSteps, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		err = executor.Execute(rec, execCtx)

		if err == nil || !strings.Contains(err.Error(), "panic in custom go step") {
			t.Fatalf("expected recovered panic error, got: %v", err)
		}
	})

	t.Run("Fails cleanly on unregistered Go step name", func(t *testing.T) {
		t.Parallel()

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "missing",
				Go:   &parser.GoStepBlock{Use: "non.existent.func"},
			},
		}

		executor := engine.NewPipelineExecutor(steps, nil, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		err = executor.Execute(rec, execCtx)

		if err == nil || !strings.Contains(err.Error(), "unregistered go function") {
			t.Fatalf("expected unregistered function error, got: %v", err)
		}
	})
}

func TestPipeline_Starlark(t *testing.T) {
	t.Parallel()

	t.Run("Executes Starlark script and reads prior step outputs", func(t *testing.T) {
		t.Parallel()

		goSteps := map[string]core.StepHandler{
			"fetch.user": func(ctx context.Context, step *core.Step) (any, error) {
				return map[string]any{
					"name":  "jane",
					"roles": []any{"admin", "editor"},
				}, nil
			},
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "auth",
				Go:   &parser.GoStepBlock{Use: "fetch.user"},
			},
			{
				Type: parser.StepTypeStarlark,
				Name: "transform",
				Starlark: &parser.StarlarkStepBlock{
					Source: `
def execute(ctx):
    user = ctx.steps.auth.get("result", {})
    roles = user.get("roles", [])
    return {
        "display_name": user.get("name", "").capitalize(),
        "total_roles": len(roles),
        "is_admin": "admin" in roles
    }
`,
				},
			},
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Status: parseExpr(t, `200`),
					Body:   parseExpr(t, `steps.transform.result`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, goSteps, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, execCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)

		if body["display_name"] != "Jane" || body["total_roles"] != float64(2) || body["is_admin"] != true {
			t.Errorf("unexpected Starlark output: %+v", body)
		}
	})

	t.Run("Halts execution safely when Starlark step limit is exceeded", func(t *testing.T) {
		t.Parallel()

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeStarlark,
				Name: "infinite_loop",
				Starlark: &parser.StarlarkStepBlock{
					Source: `
def execute(ctx):
    x = 0
    while True:
        x += 1
    return x
`,
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, nil, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		err = executor.Execute(rec, execCtx)

		if err == nil || !strings.Contains(err.Error(), "too many steps") {
			t.Fatalf("expected step limit exceeded error, got: %v", err)
		}
	})
}

func TestPipeline_SQL(t *testing.T) {
	t.Parallel()

	t.Run("Executes query and exports row, rows, and rows_affected", func(t *testing.T) {
		t.Parallel()

		mgr := setupTestSQLiteManager(t)
		pool, _ := mgr.Get("sqlite.main")

		createStmt := `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);`
		if _, err := pool.DB.ExecContext(t.Context(), createStmt); err != nil {
			t.Fatalf("failed to create users table: %v", err)
		}

		insertStmt1 := `INSERT INTO users VALUES (1, 'Jane', 'jane@example.com');`
		if _, err := pool.DB.ExecContext(t.Context(), insertStmt1); err != nil {
			t.Fatalf("failed to insert user 1: %v", err)
		}

		insertStmt2 := `INSERT INTO users VALUES (2, 'John', 'john@example.com');`
		if _, err := pool.DB.ExecContext(t.Context(), insertStmt2); err != nil {
			t.Fatalf("failed to insert user 2: %v", err)
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeSQL,
				Name: "find_user",
				SQL: &parser.SQLStepBlock{
					Connection: parseExpr(t, `connection.sqlite.main`),
					Query:      "SELECT id, name, email FROM users WHERE id = @id",
					Args:       parseExpr(t, `{ id = ctx.request.path.id }`),
				},
			},
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Condition: parseExpr(t, `steps.find_user.rows_affected > 0`),
					Status:    parseExpr(t, `200`),
					Body: parseExpr(t, `{
						user          = steps.find_user.row
						all_users     = steps.find_user.rows
						total_matched = steps.find_user.rows_affected
					}`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, nil, mgr)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/1", http.NoBody)
		req.SetPathValue("id", "1")

		execCtx, err := core.NewExecutionContext(nil, req, core.WithPathParams([]string{"id"}))
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, execCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&resp)

		if resp["total_matched"] != float64(1) {
			t.Errorf("expected total_matched 1, got %v", resp["total_matched"])
		}

		user, ok := resp["user"].(map[string]any)
		if !ok || user["name"] != "Jane" || user["email"] != "jane@example.com" {
			t.Errorf("expected user Jane, got %+v", resp["user"])
		}
	})

	t.Run("Intercepts constraint violation with catch block", func(t *testing.T) {
		t.Parallel()

		mgr := setupTestSQLiteManager(t)
		pool, _ := mgr.Get("sqlite.main")

		schema := `
			CREATE TABLE accounts (id INTEGER PRIMARY KEY, email TEXT UNIQUE);
			INSERT INTO accounts VALUES (1, 'existing@example.com');
		`
		if _, err := pool.DB.ExecContext(t.Context(), schema); err != nil {
			t.Fatalf("failed to seed test table: %v", err)
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeSQL,
				Name: "insert_account",
				SQL: &parser.SQLStepBlock{
					Connection: parseExpr(t, `connection.sqlite.main`),
					Query:      "INSERT INTO accounts (id, email) VALUES (2, @email)",
					Args:       parseExpr(t, `{ email = ctx.request.body.email }`),
					Catches: []parser.SQLCatchBlock{
						{
							Code:    "19",
							Status:  parseExpr(t, `409`),
							Headers: parseExpr(t, `{ "X-Error" = "Conflict" }`),
							Body:    parseExpr(t, `{ error = "Account with this email already exists" }`),
						},
					},
				},
			},
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Status: parseExpr(t, `201`),
					Body:   parseExpr(t, `steps.insert_account.row`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, nil, mgr)

		req := httptest.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			"/accounts",
			strings.NewReader(`{"email": "existing@example.com"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, execCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusConflict {
			t.Errorf("expected status 409 Conflict, got %d. Body: %s", rec.Code, rec.Body.String())
		}
		if h := rec.Header().Get("X-Error"); h != "Conflict" {
			t.Errorf("expected header X-Error 'Conflict', got %q", h)
		}
	})
}

func TestPipeline_Respond(t *testing.T) {
	t.Parallel()

	t.Run("Evaluates custom headers and body payload", func(t *testing.T) {
		t.Parallel()

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Status: parseExpr(t, `200`),
					Headers: parseExpr(t, `{
						"X-Custom-Header" = "custom_value"
						"X-Echo-Method"   = ctx.request.method
					}`),
					Body: parseExpr(t, `{ ok = true }`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, nil, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, execCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if h := rec.Header().Get("X-Custom-Header"); h != "custom_value" {
			t.Errorf("expected header 'custom_value', got %q", h)
		}
		if h := rec.Header().Get("X-Echo-Method"); h != "GET" {
			t.Errorf("expected header 'GET', got %q", h)
		}
	})

	t.Run("Omitted body produces an empty response payload", func(t *testing.T) {
		t.Parallel()

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeRespond,
				Respond: &parser.RespondStepBlock{
					Status: parseExpr(t, `204`),
				},
			},
		}

		executor := engine.NewPipelineExecutor(steps, nil, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := executor.Execute(rec, execCtx); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("expected empty body, got %q", rec.Body.String())
		}
	})
}

func TestPipeline_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("Aborts pipeline immediately when request context is canceled", func(t *testing.T) {
		t.Parallel()

		stepTwoExecuted := false
		goSteps := map[string]core.StepHandler{
			"step.one": func(ctx context.Context, step *core.Step) (any, error) {
				return "done_one", nil
			},
			"step.two": func(ctx context.Context, step *core.Step) (any, error) {
				stepTwoExecuted = true
				return "done_two", nil
			},
		}

		steps := []parser.ParsedStep{
			{
				Type: parser.StepTypeGo,
				Name: "first",
				Go:   &parser.GoStepBlock{Use: "step.one"},
			},
			{
				Type: parser.StepTypeGo,
				Name: "second",
				Go:   &parser.GoStepBlock{Use: "step.two"},
			},
		}

		executor := engine.NewPipelineExecutor(steps, goSteps, nil)

		reqCtx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel before execution

		req := httptest.NewRequestWithContext(reqCtx, http.MethodGet, "/test", http.NoBody)
		execCtx, err := core.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		err = executor.Execute(rec, execCtx)

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got: %v", err)
		}
		if stepTwoExecuted {
			t.Errorf("expected step two to be skipped after context cancellation")
		}
	})
}
