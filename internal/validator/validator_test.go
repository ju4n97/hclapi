package validator_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/validator"
)

func TestValidateBody(t *testing.T) {
	t.Parallel()

	fields := []core.Field{
		{
			Name:     "email",
			Type:     "string",
			Required: true,
			Format:   "email",
		},
		{
			Name:      "username",
			Type:      "string",
			Required:  true,
			MinLength: new(3),
			MaxLength: new(10),
			Pattern:   "^[a-z0-9_]+$",
		},
		{
			Name:     "role",
			Type:     "string",
			Required: false,
			Default:  "member",
			Enum:     []any{"admin", "member", "viewer"},
		},
		{
			Name:     "age",
			Type:     "int",
			Required: false,
			Min:      new(float64(18)),
			Max:      new(float64(100)),
		},
		{
			Name:     "bio",
			Type:     "string",
			Required: false,
		},
		{
			Name:        "tags",
			Type:        "list(string)",
			Required:    false,
			MinItems:    new(1),
			MaxItems:    new(3),
			UniqueItems: true,
		},
	}

	t.Run("Valid payload passes, injects defaults, and normalizes optional missing fields to nil", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"email":    "jane@example.com",
			"username": "jane_doe",
			"age":      25,
			"tags":     []any{"go", "api"},
		}

		normalized, errs := validator.ValidateBody(data, fields)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got %d: %+v", len(errs), errs)
		}

		// Defaults injected
		if normalized["role"] != "member" {
			t.Errorf("expected default role 'member', got %v", normalized["role"])
		}
		// Optional missing fields normalized to explicit nil
		if normalized["bio"] != nil {
			t.Errorf("expected missing optional field 'bio' to be nil, got %v", normalized["bio"])
		}
	})

	t.Run("Catches missing required fields", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"username": "jane_doe",
		}

		_, errs := validator.ValidateBody(data, fields)
		if len(errs) != 1 || errs[0].Name != "email" || errs[0].Reason != "field is required" {
			t.Fatalf("expected required email error, got: %+v", errs)
		}
	})

	t.Run("Catches invalid format, bounds, pattern, enum, and duplicates", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"email":    "invalid-email",
			"username": "AB!",
			"role":     "superadmin",
			"age":      15,
			"tags":     []any{"go", "go"},
		}

		_, errs := validator.ValidateBody(data, fields)
		if len(errs) != 5 {
			t.Fatalf("expected 5 errors, got %d: %+v", len(errs), errs)
		}
	})
}

func TestValidateStringMap(t *testing.T) {
	t.Parallel()

	fields := []core.Field{
		{Name: "page", Type: "int", Default: 1},
		{Name: "limit", Type: "int", Required: true, Default: int64(25), Min: new(float64(1)), Max: new(float64(100))},
		{Name: "sort", Type: "string", Enum: []any{"asc", "desc"}},
		{Name: "active", Type: "bool"},
	}

	t.Run("Validates and injects defaults in-place", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{
			"sort":   "desc",
			"active": "true",
		}

		errs := validator.ValidateStringMap(data, fields)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got: %+v", errs)
		}

		if data["page"] != "1" {
			t.Errorf("expected default page '1', got %q", data["page"])
		}
		if data["limit"] != "25" {
			t.Errorf("expected default limit '25', got %q", data["limit"])
		}
	})

	t.Run("Catches invalid coerced types and bounds in string map", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{
			"limit":  "200", // exceeds max 100
			"sort":   "random",
			"active": "invalid",
		}

		errs := validator.ValidateStringMap(data, fields)
		if len(errs) != 3 {
			t.Fatalf("expected 3 errors, got %d: %+v", len(errs), errs)
		}
	})
}

func TestValidateHeaders(t *testing.T) {
	t.Parallel()

	fields := []core.Field{
		{Name: "Authorization", Type: "string", Required: true},
		{Name: "X-Trace-Sampled", Type: "bool", Default: true},
	}

	t.Run("Case-insensitive lookup against lowercased headers with default injection", func(t *testing.T) {
		t.Parallel()

		headers := map[string]string{
			"authorization": "Bearer token", // Ingress lowercases all headers
		}

		errs := validator.ValidateHeaders(headers, fields)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got: %+v", errs)
		}

		if headers["x-trace-sampled"] != "true" {
			t.Errorf("expected injected default 'true', got %q", headers["x-trace-sampled"])
		}
	})

	t.Run("Missing required header retains schema casing in error diagnostic", func(t *testing.T) {
		t.Parallel()

		headers := map[string]string{}

		errs := validator.ValidateHeaders(headers, fields)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Name != "Authorization" {
			t.Errorf("expected error field name 'Authorization', got %q", errs[0].Name)
		}
	})
}
