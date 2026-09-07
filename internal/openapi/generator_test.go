package openapi_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/openapi"
	"github.com/ju4n97/hclapi/internal/service"
)

func TestOpenAPI_ComprehensiveGeneration(t *testing.T) {
	t.Parallel()

	svc := &service.Definition{
		Server: service.Server{
			MaxBodySize: 10 * 1024 * 1024,
		},
		OpenAPI: service.OpenAPI{
			Title:       "Acme Store API",
			Version:     "1.0.0",
			Description: "Comprehensive API specification for testing.",
			Servers: []service.OpenAPIServer{
				{URL: "https://api.example.com/v1", Description: "Production"},
				{URL: "http://localhost:8080", Description: "Local"},
			},
			Tags: []service.OpenAPITag{
				{Name: "users", Description: "User account management"},
			},
			Contact: &service.Contact{
				Name:  "API Support",
				Email: "support@example.com",
				URL:   "https://example.com/support",
			},
			License: &service.License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Schemas: map[string]service.Schema{
			"user": {
				Name: "user",
				Fields: []service.Field{
					{Name: "email", Type: "string", Required: true, Format: "email", Description: "User email"},
					{Name: "role", Type: "string", Default: "member", Enum: []any{"admin", "member"}},
				},
			},
		},
		Endpoints: []service.Endpoint{
			{
				RoutePattern: "POST /api/v1/users/{id}",
				Method:       "POST",
				Path:         "/api/v1/users/{id}",
				Description:  "Registers a new user record.",
				RequestRules: service.RequestRules{
					PathFields: []service.Field{
						{Name: "id", Type: "int", Required: true, Description: "Unique user ID"},
					},
					HeaderFields: []service.Field{
						{Name: "x-api-key", Type: "string", Required: true, Format: "uuid"},
					},
					QueryFields: []service.Field{
						{Name: "source", Type: "string", Default: "direct"},
					},
					BodyFields: []service.Field{
						{Name: "email", Type: "string", Required: true, Format: "email"},
					},
				},
				Handler: service.PipelineHandler{
					Steps: []service.Step{
						{
							Type:    service.StepTypeRespond,
							Respond: &service.RespondStep{},
						},
					},
				},
			},
			{
				RoutePattern: "GET /static/{filepath...}",
				Method:       "GET",
				Path:         "/static/{filepath...}",
				Description:  "Serves public assets.",
				RequestRules: service.RequestRules{
					PathFields: []service.Field{
						{Name: "filepath", Type: "string", Required: true},
					},
				},
				Handler: service.PipelineHandler{
					Steps: []service.Step{
						{
							Type:    service.StepTypeRespond,
							Respond: &service.RespondStep{},
						},
					},
				},
			},
			{
				RoutePattern: "GET /docs",
				Method:       "GET",
				Path:         "/docs",
				Handler: service.OpenAPIHandler{
					Mode:     "ui",
					Renderer: "scalar",
					SpecURL:  "/openapi.json",
				},
			},
			{
				RoutePattern: "GET /openapi.json",
				Method:       "GET",
				Path:         "/openapi.json",
				Handler: service.OpenAPIHandler{
					Mode:   "spec",
					Format: "json",
				},
			},
		},
	}

	t.Run("Generates fully verified OpenAPI 3.1 JSON document", func(t *testing.T) {
		t.Parallel()

		jsonBytes, err := openapi.GenerateJSON(svc, true)
		if err != nil {
			t.Fatalf("unexpected generation error: %v", err)
		}

		var doc map[string]any
		if err := json.Unmarshal(jsonBytes, &doc); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if doc["openapi"] != "3.1.0" {
			t.Errorf("expected openapi '3.1.0', got %v", doc["openapi"])
		}
		info := doc["info"].(map[string]any)
		if info["title"] != "Acme Store API" || info["version"] != "1.0.0" {
			t.Errorf("unexpected info metadata: %+v", info)
		}

		components := doc["components"].(map[string]any)
		schemas := components["schemas"].(map[string]any)
		userSchema := schemas["user"].(map[string]any)
		userProps := userSchema["properties"].(map[string]any)
		if userProps["email"] == nil || userProps["role"] == nil {
			t.Errorf("expected email and role in user schema: %+v", userProps)
		}

		paths := doc["paths"].(map[string]any)
		if paths["/api/v1/users/{id}"] == nil {
			t.Fatalf("expected path '/api/v1/users/{id}' in document: %+v", paths)
		}

		if paths["/static/{filepath}"] == nil {
			t.Errorf("expected catch-all path '/static/{filepath}' in document: %+v", paths)
		}

		if paths["/docs"] != nil {
			t.Errorf("expected /docs to be excluded from paths catalog")
		}
		if paths["/openapi.json"] != nil {
			t.Errorf("expected /openapi.json to be excluded from paths catalog")
		}

		userPathItem := paths["/api/v1/users/{id}"].(map[string]any)
		postOp := userPathItem["post"].(map[string]any)
		params := postOp["parameters"].([]any)
		if len(params) != 3 {
			t.Errorf("expected 3 parameters (path, header, query), got %d: %+v", len(params), params)
		}
		if postOp["requestBody"] == nil {
			t.Errorf("expected requestBody on POST operation")
		}

		responses := postOp["responses"].(map[string]any)
		for _, code := range []string{"200", "413", "422", "500"} {
			if responses[code] == nil {
				t.Errorf("expected response code %q, got: %+v", code, responses)
			}
		}
	})

	t.Run("Generates valid OpenAPI 3.1 YAML document", func(t *testing.T) {
		t.Parallel()

		yamlBytes, err := openapi.GenerateYAML(svc)
		if err != nil {
			t.Fatalf("unexpected YAML generation error: %v", err)
		}

		yamlStr := string(yamlBytes)
		if !strings.Contains(yamlStr, "openapi: 3.1.0") {
			t.Errorf("expected openapi: 3.1.0 header in YAML")
		}
		if !strings.Contains(yamlStr, "title: Acme Store API") {
			t.Errorf("expected title in YAML")
		}
	})
}
