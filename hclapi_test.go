package hclapi_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi"
)

// testHelper creates a temporary manifest directory and boots a Hclapi engine.
func setupTestEngine(t *testing.T, hclManifest string, registerSteps func(*hclapi.Engine)) *hclapi.Engine {
	t.Helper()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "Hclapifile"), []byte(hclManifest), 0644); err != nil {
		t.Fatalf("setupTestEngine: failed to write Hclapifile: %v", err)
	}

	engine, err := hclapi.NewEngine(hclapi.Options{
		ManifestDir:  tmpDir,
		StrictTyping: true,
	})
	if err != nil {
		t.Fatalf("setupTestEngine: failed to create engine: %v", err)
	}

	if registerSteps != nil {
		registerSteps(engine)
	}

	return engine
}

func TestEngine_ContextExtraction(t *testing.T) {
	t.Parallel()

	manifest := `
		endpoint "POST /api/v1/inspect/{tenant_id}/users/{user_id}" {
			pipeline {
				go "extractor" {
					use = "test.extract_all"
				}
				respond {
					status = 200
				}
			}
		}
	`

	engine := setupTestEngine(t, manifest, func(e *hclapi.Engine) {
		_ = e.RegisterStep("test.extract_all", func(ctx *hclapi.Context) (any, error) {
			return map[string]any{
				"path_tenant": ctx.Request.Path["tenant_id"],
				"path_user":   ctx.Request.Path["user_id"],
				"query_sort":  ctx.Request.Query["sort"],
				"header_auth": ctx.Request.Headers["authorization"],
				"body":        ctx.Request.Body,
			}, nil
		})
	})

	req := httptest.NewRequest("POST", "/api/v1/inspect/t_99/users/u_101?sort=desc", strings.NewReader(`{"role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret_jwt_token")

	w := httptest.NewRecorder()
	engine.Handler().ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expectedJSON := `{"body":{"role":"admin"},"header_auth":"Bearer secret_jwt_token","path_tenant":"t_99","path_user":"u_101","query_sort":"desc"}`

	if strings.TrimSpace(string(body)) != expectedJSON {
		t.Errorf("expected body %s, got %s", expectedJSON, string(body))
	}
}

func TestEngine_StepChaining(t *testing.T) {
	t.Parallel()

	manifest := `
		endpoint "POST /api/v1/calculate" {
			pipeline {
				go "step_one" {
					use = "math.double"
				}
				go "step_two" {
					use = "math.add_ten"
				}
				respond {
					status = 200
				}
			}
		}
	`

	engine := setupTestEngine(t, manifest, func(e *hclapi.Engine) {
		// Step 1: Take input number and double it
		_ = e.RegisterStep("math.double", func(ctx *hclapi.Context) (any, error) {
			bodyMap := ctx.Request.Body.(map[string]any)
			val := bodyMap["value"].(float64)
			return map[string]any{"doubled": val * 2}, nil
		})

		// Step 2: Read Step 1 output from ctx.Steps and add 10
		_ = e.RegisterStep("math.add_ten", func(ctx *hclapi.Context) (any, error) {
			stepOneResult := ctx.Steps["step_one"].Result.(map[string]any)
			doubled := stepOneResult["doubled"].(float64)
			return map[string]any{"final_result": doubled + 10}, nil
		})
	})

	req := httptest.NewRequest("POST", "/api/v1/calculate", strings.NewReader(`{"value": 5}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.Handler().ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expectedJSON := `{"final_result":20}`

	if strings.TrimSpace(string(body)) != expectedJSON {
		t.Errorf("expected body %s, got %s", expectedJSON, string(body))
	}
}

func TestEngine_ErrorHandling(t *testing.T) {
	t.Parallel()

	manifest := `
		endpoint "GET /api/v1/unregistered" {
			pipeline {
				go "missing" {
					use = "does.not.exist"
				}
				respond {
					status = 200
				}
			}
		}

		endpoint "GET /api/v1/fails" {
			pipeline {
				go "broken" {
					use = "test.failing_step"
				}
				respond {
					status = 200
				}
			}
		}
	`

	engine := setupTestEngine(t, manifest, func(e *hclapi.Engine) {
		_ = e.RegisterStep("test.failing_step", func(ctx *hclapi.Context) (any, error) {
			return nil, errors.New("database connection refused")
		})
	})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Unregistered Go step returns 500",
			path:           "/api/v1/unregistered",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `unregistered go function \"does.not.exist\"`,
		},
		{
			name:           "Failing Go step returns 500 with error details",
			path:           "/api/v1/fails",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error": "database connection refused"}`,
		},
		{
			name:           "Non-existent endpoint returns 404",
			path:           "/api/v1/not-found",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			engine.Handler().ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			trimmed := strings.TrimSpace(string(body))
			if !strings.Contains(trimmed, strings.TrimSpace(tt.expectedBody)) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, trimmed)
			}
		})
	}
}

func TestEngine_LifecycleGuards(t *testing.T) {
	t.Parallel()

	t.Run("Duplicate step registration returns error", func(t *testing.T) {
		t.Parallel()

		manifest := `
			endpoint "GET /a" {
				pipeline {
					respond {
						status = 200
					}
				}
			}	
		`

		engine := setupTestEngine(t, manifest, nil)

		err := engine.RegisterStep("duplicate.step", func(ctx *hclapi.Context) (any, error) { return nil, nil })
		if err != nil {
			t.Fatalf("first registration should succeed: %v", err)
		}

		err = engine.RegisterStep("duplicate.step", func(ctx *hclapi.Context) (any, error) { return nil, nil })
		if err == nil {
			t.Fatalf("expected error on duplicate step registration, got nil")
		}
	})

	t.Run("Invalid manifest directory returns error", func(t *testing.T) {
		t.Parallel()

		_, err := hclapi.NewEngine(hclapi.Options{
			ManifestDir: "non_existent_directory_xyz",
		})
		if err == nil {
			t.Fatalf("expected error when booting with non-existent directory, got nil")
		}
	})
}

func TestEngine_StarlarkPipeline(t *testing.T) {
	t.Parallel()

	manifest := `
		endpoint "POST /api/v1/normalize" {
			pipeline {
				starlark "prep" {
					source = <<-STARLARK
						def execute(ctx):
						    raw_tags = ctx.request.body.tags
						    cleaned_tags = [t.strip().lower() for t in raw_tags if len(t.strip()) > 0]
						    return {
						        "username": ctx.request.body.name.strip().upper(),
						        "tag_count": len(cleaned_tags),
						        "tags": cleaned_tags
						    }
					STARLARK
				}
				respond {
					status = 200
				}
			}
		}

		endpoint "GET /api/v1/starlark-error" {
			pipeline {
				starlark "bad" {
					source = <<-STARLARK
						def execute(ctx):
						    return 10 / 0 # Runtime division by zero
					STARLARK
				}
				respond {
					status = 200
				}
			}
		}
	`

	engine := setupTestEngine(t, manifest, nil)

	t.Run("Valid Starlark Transformation", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("POST", "/api/v1/normalize", strings.NewReader(`{"name":"  jane  ","tags":[" Golang ",""," PYTHON "]}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.Handler().ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := `{"tag_count":2,"tags":["golang","python"],"username":"JANE"}`
		if strings.TrimSpace(string(body)) != expected {
			t.Errorf("expected %s, got %s", expected, strings.TrimSpace(string(body)))
		}
	})

	t.Run("Starlark Runtime Error Returns 500", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/api/v1/starlark-error", nil)
		w := httptest.NewRecorder()

		engine.Handler().ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", resp.StatusCode)
		}
	})
}
