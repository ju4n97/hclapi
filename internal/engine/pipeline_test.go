package engine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/ju4n97/hclapi/internal/engine"
	"github.com/ju4n97/hclapi/internal/runtime"
	"github.com/ju4n97/hclapi/internal/service"
	"github.com/ju4n97/hclapi/internal/sqldb"
)

func parseExpr(t *testing.T, src string) hcl.Expression {
	t.Helper()
	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("syntax error in expression %q: %s", src, diags.Error())
	}
	return expr
}

func setupSQLitePool(t *testing.T) *sqldb.Pool {
	t.Helper()

	mgr := sqldb.NewManager()
	cfg := sqldb.Config{
		Driver: "sqlite",
		Name:   "main",
		Source: "file:pipeline_test_mem?mode=memory&cache=shared",
		Pool:   sqldb.DefaultPoolConfig(),
	}
	if err := mgr.Open(t.Context(), cfg); err != nil {
		t.Fatalf("failed to open sqlite pool: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	pool, _ := mgr.Get("sqlite.main")
	return pool
}

func TestPipeline_Execution(t *testing.T) {
	t.Parallel()

	t.Run("Executes GoStep via StepRegistry and exports result", func(t *testing.T) {
		t.Parallel()

		registry := engine.NewStepRegistry()
		_ = registry.Register("auth.verify", func(ctx context.Context, step *runtime.Step) (any, error) {
			token := step.Args.GetOr("token", "")
			return map[string]any{
				"valid": token == "secret-token",
				"uid":   42,
			}, nil
		})

		pipeline := engine.NewPipeline(
			&engine.GoStep{
				Name:     "auth",
				Use:      "auth.verify",
				Registry: registry,
				Args:     parseExpr(t, `{ token = ctx.request.headers.authorization }`),
			},
			&engine.RespondStep{
				Status: parseExpr(t, `200`),
				Body:   parseExpr(t, `steps.auth.result`),
			},
		)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		req.Header.Set("Authorization", "secret-token")

		execCtx, err := runtime.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := pipeline.Execute(execCtx, rec); err != nil {
			t.Fatalf("execution error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["valid"] != true || body["uid"] != float64(42) {
			t.Errorf("unexpected body: %+v", body)
		}
	})

	t.Run("Recovers from panic in GoStep safely", func(t *testing.T) {
		t.Parallel()

		registry := engine.NewStepRegistry()
		_ = registry.Register("panic.step", func(ctx context.Context, step *runtime.Step) (any, error) {
			panic("nil pointer dereference inside user step")
		})

		pipeline := engine.NewPipeline(
			&engine.GoStep{
				Name:     "broken",
				Use:      "panic.step",
				Registry: registry,
			},
		)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, err := runtime.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}

		rec := httptest.NewRecorder()
		err = pipeline.Execute(execCtx, rec)

		if err == nil || !strings.Contains(err.Error(), "panic in custom go step") {
			t.Fatalf("expected recovered panic error, got: %v", err)
		}
	})

	t.Run("Executes StarlarkStep and transforms prior outputs", func(t *testing.T) {
		t.Parallel()

		registry := engine.NewStepRegistry()
		_ = registry.Register("fetch", func(ctx context.Context, step *runtime.Step) (any, error) {
			return map[string]any{
				"name":  "jane",
				"roles": []any{"admin", "editor"},
			}, nil
		})

		pipeline := engine.NewPipeline(
			&engine.GoStep{
				Name:     "fetch",
				Use:      "fetch",
				Registry: registry,
			},
			&engine.StarlarkStep{
				Name: "transform",
				Source: `
def execute(ctx):
    user = ctx.steps.fetch["result"]
    return {
        "display": user["name"].capitalize(),
        "is_admin": "admin" in user["roles"]
    }
`,
			},
			&engine.RespondStep{
				Status: parseExpr(t, `200`),
				Body:   parseExpr(t, `steps.transform.result`),
			},
		)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		execCtx, _ := runtime.NewExecutionContext(nil, req)
		rec := httptest.NewRecorder()

		if err := pipeline.Execute(execCtx, rec); err != nil {
			t.Fatalf("execution error: %v", err)
		}

		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["display"] != "Jane" || body["is_admin"] != true {
			t.Errorf("unexpected body: %+v", body)
		}
	})

	t.Run("Executes SQLStep and intercepts constraint catch block", func(t *testing.T) {
		t.Parallel()

		pool := setupSQLitePool(t)

		schema := `
			DROP TABLE IF EXISTS accounts;
			CREATE TABLE accounts (id INTEGER PRIMARY KEY, email TEXT UNIQUE);
			INSERT INTO accounts VALUES (1, 'existing@example.com');
		`
		if _, err := pool.DB.ExecContext(t.Context(), schema); err != nil {
			t.Fatalf("failed to seed test table: %v", err)
		}

		pipeline := engine.NewPipeline(
			&engine.SQLStep{
				Name:  "insert",
				Pool:  pool,
				Query: "INSERT INTO accounts (id, email) VALUES (2, @email)",
				Args:  parseExpr(t, `{ email = ctx.request.body.email }`),
				Catches: []service.SQLCatch{
					{
						Code:    "19",
						Status:  parseExpr(t, `409`),
						Headers: parseExpr(t, `{ "X-Conflict" = "true" }`),
						Body:    parseExpr(t, `{ error = "Account with this email already exists" }`),
					},
				},
			},
			&engine.RespondStep{
				Status: parseExpr(t, `201`),
				Body:   parseExpr(t, `steps.insert.row`),
			},
		)

		req := httptest.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			"/accounts",
			strings.NewReader(`{"email": "existing@example.com"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		execCtx, err := runtime.NewExecutionContext(nil, req)
		if err != nil {
			t.Fatalf("failed to create context: %v", err)
		}
		rec := httptest.NewRecorder()

		if err := pipeline.Execute(execCtx, rec); err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}

		if rec.Code != http.StatusConflict {
			t.Errorf("expected status 409 Conflict, got %d. Body: %s", rec.Code, rec.Body.String())
		}
		if h := rec.Header().Get("X-Conflict"); h != "true" {
			t.Errorf("expected header X-Conflict 'true', got %q", h)
		}
	})
}
