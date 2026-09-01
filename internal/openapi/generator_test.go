package openapi_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/compiler"
	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/openapi"
	"github.com/ju4n97/hclapi/internal/parser"
)

func TestOpenAPI_ComprehensiveGeneration(t *testing.T) {
	t.Parallel()

	service := &compiler.CompiledService{
		Server: core.Server{
			MaxBodySize: core.ByteSize(10 * 1024 * 1024),
			OpenAPI: core.OpenAPIConfig{
				Title:       "Acme Store API",
				Version:     "1.0.0",
				Description: "Comprehensive API specification for testing.",
				Servers: []core.OpenAPIServer{
					{URL: "https://api.example.com/v1", Description: "Production"},
					{URL: "http://localhost:8080", Description: "Local"},
				},
				Tags: []core.OpenAPITag{
					{Name: "users", Description: "User account management"},
					{Name: "orders", Description: "Order processing"},
				},
				Contact: &core.OpenAPIContact{
					Name:  "API Support",
					Email: "support@example.com",
					URL:   "https://example.com/support",
				},
				License: &core.OpenAPILicense{
					Name: "MIT",
					URL:  "https://opensource.org/licenses/MIT",
				},
			},
		},
		Schemas: map[string][]core.Field{
			"user": {
				{Name: "email", Type: "string", Required: true, Format: "email", Description: "User email"},
				{
					Name:      "username",
					Type:      "string",
					Required:  true,
					MinLength: new(3),
					MaxLength: new(20),
					Pattern:   "^[a-z0-9_]+$",
				},
				{Name: "role", Type: "string", Default: "member", Enum: []any{"admin", "member", "viewer"}},
				{Name: "age", Type: "int", Min: new(float64(18)), Max: new(float64(100))},
				{Name: "tags", Type: "list(string)", MinItems: new(1), MaxItems: new(5), UniqueItems: true},
			},
		},
		Endpoints: []compiler.CompiledEndpoint{
			{
				MethodAndPath: "POST /api/v1/users/{id}",
				Description:   "Registers a new user record.",
				Rules: compiler.CompiledRequestRules{
					PathFields: []core.Field{
						{Name: "id", Type: "int", Required: true, Description: "Unique user ID"},
					},
					HeaderFields: []core.Field{
						{Name: "x-api-key", Type: "string", Required: true, Format: "uuid"},
					},
					QueryFields: []core.Field{
						{Name: "source", Type: "string", Default: "direct", Enum: []any{"direct", "referral"}},
					},
					BodyFields: []core.Field{
						{Name: "email", Type: "string", Required: true, Format: "email"},
						{Name: "username", Type: "string", Required: true, MinLength: new(3)},
					},
				},
				Steps: []parser.ParsedStep{
					{
						Type: parser.StepTypeSQL,
						Name: "insert_user",
						SQL: &parser.SQLStepBlock{
							Catches: []parser.SQLCatchBlock{
								{Code: "23505", Status: nil}, // Implies 409 Conflict
							},
						},
					},
					{
						Type: parser.StepTypeRespond,
						Respond: &parser.RespondStepBlock{
							Status: nil, // Implies 201 / 200
						},
					},
				},
			},
			{
				MethodAndPath: "GET /static/{filepath...}",
				Description:   "Serves public assets.",
				Rules: compiler.CompiledRequestRules{
					PathFields: []core.Field{
						{Name: "filepath", Type: "string", Required: true},
					},
				},
				Steps: []parser.ParsedStep{
					{
						Type: parser.StepTypeRespond,
						Respond: &parser.RespondStepBlock{
							Status: nil,
						},
					},
				},
			},
			{
				MethodAndPath: "GET /docs",
				OpenAPI: &compiler.CompiledOpenAPIHandler{
					UI: "scalar",
				},
			},
		},
	}

	t.Run("Generates fully verified OpenAPI 3.1 JSON document", func(t *testing.T) {
		t.Parallel()

		jsonBytes, err := openapi.GenerateJSON(service, true)
		if err != nil {
			t.Fatalf("unexpected generation error: %v", err)
		}

		var doc map[string]any
		if err := json.Unmarshal(jsonBytes, &doc); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		// Verify OpenAPI Header and Info
		if doc["openapi"] != "3.1.0" {
			t.Errorf("expected openapi '3.1.0', got %v", doc["openapi"])
		}
		info := doc["info"].(map[string]any)
		if info["title"] != "Acme Store API" || info["version"] != "1.0.0" {
			t.Errorf("unexpected info metadata: %+v", info)
		}

		// Verify Components / Reusable Schemas
		components := doc["components"].(map[string]any)
		schemas := components["schemas"].(map[string]any)
		userSchema := schemas["user"].(map[string]any)
		userProps := userSchema["properties"].(map[string]any)
		if userProps["email"] == nil || userProps["role"] == nil {
			t.Errorf("expected email and role properties in user schema: %+v", userProps)
		}

		// Verify Paths
		paths := doc["paths"].(map[string]any)
		if paths["/api/v1/users/{id}"] == nil {
			t.Fatalf("expected path '/api/v1/users/{id}' in document: %+v", paths)
		}

		// Verify Catch-all wildcard converted from {filepath...} to {filepath}
		if paths["/static/{filepath}"] == nil {
			t.Errorf("expected catch-all path '/static/{filepath}' in document: %+v", paths)
		}

		// Verify docs endpoint was excluded from API routes
		if paths["/docs"] != nil {
			t.Errorf("expected docs endpoint to be excluded from API routes")
		}

		// Verify Operation Parameters and Request Body
		userPathItem := paths["/api/v1/users/{id}"].(map[string]any)
		postOp := userPathItem["post"].(map[string]any)
		params := postOp["parameters"].([]any)
		if len(params) != 3 { // path: id, header: x-api-key, query: source
			t.Errorf("expected 3 parameters (path, header, query), got %d: %+v", len(params), params)
		}

		if postOp["requestBody"] == nil {
			t.Errorf("expected requestBody on POST operation")
		}

		// Verify Responses (200/201, 409 Conflict, 413 Payload Too Large, 422 Unprocessable Entity, 500)
		responses := postOp["responses"].(map[string]any)
		expectedCodes := []string{"200", "422", "500"}
		for _, code := range expectedCodes {
			if responses[code] == nil {
				t.Errorf("expected response code %q on operation, got: %+v", code, responses)
			}
		}
	})

	t.Run("Generates valid OpenAPI 3.1 YAML document", func(t *testing.T) {
		t.Parallel()

		yamlBytes, err := openapi.GenerateYAML(service)
		if err != nil {
			t.Fatalf("unexpected YAML generation error: %v", err)
		}

		yamlStr := string(yamlBytes)
		if !strings.Contains(yamlStr, "openapi: 3.1.0") {
			t.Errorf("expected openapi: 3.1.0 header in YAML output")
		}
		if !strings.Contains(yamlStr, "title: Acme Store API") {
			t.Errorf("expected title in YAML output")
		}
		if !strings.Contains(yamlStr, "/api/v1/users/{id}:") {
			t.Errorf("expected path in YAML output")
		}
	})
}
