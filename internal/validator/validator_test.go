package validator_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/validator"
)

func TestValidate(t *testing.T) {
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
			Name:        "tags",
			Type:        "list(string)",
			Required:    false,
			MinItems:    new(1),
			MaxItems:    new(3),
			UniqueItems: true,
		},
	}

	t.Run("Valid payload passes with 0 invalid params", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"email":    "jane@example.com",
			"username": "jane_doe",
			"role":     "member",
			"age":      25,
			"tags":     []any{"go", "api"},
		}

		errors := validator.Validate(data, fields)
		if len(errors) != 0 {
			t.Fatalf("expected 0 errors, got %d: %+v", len(errors), errors)
		}
	})

	t.Run("Catches missing required fields", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"username": "jane_doe",
		}

		errors := validator.Validate(data, fields)
		if len(errors) != 1 || errors[0].Name != "email" || errors[0].Reason != "field is required" {
			t.Fatalf("expected required email error, got: %+v", errors)
		}
	})

	t.Run("Rejects empty string on required field", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"email":    "   ", // Empty whitespace string
			"username": "jane_doe",
		}

		errors := validator.Validate(data, fields)
		if len(errors) != 1 || errors[0].Name != "email" {
			t.Fatalf("expected empty string error on required email, got: %+v", errors)
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

		errors := validator.Validate(data, fields)
		if len(errors) != 5 {
			t.Fatalf("expected 5 errors, got %d: %+v", len(errors), errors)
		}

		errMap := make(map[string]string, len(errors))
		for _, err := range errors {
			errMap[err.Name] = err.Reason
		}

		if errMap["email"] != "must be a valid email format" {
			t.Errorf("unexpected email error: %q", errMap["email"])
		}
		if errMap["role"] != `must be one of: ["admin", "member", "viewer"]` {
			t.Errorf("unexpected role error: %q", errMap["role"])
		}
		if errMap["age"] != "must be greater than or equal to 18" {
			t.Errorf("unexpected age error: %q", errMap["age"])
		}
		if errMap["tags"] != "items must be unique" {
			t.Errorf("unexpected tags error: %q", errMap["tags"])
		}
	})
}

func TestValidateStringMap(t *testing.T) {
	t.Parallel()

	fields := []core.Field{
		{Name: "limit", Type: "int", Required: true, Min: new(float64(1)), Max: new(float64(100))},
		{Name: "sort", Type: "string", Enum: []any{"asc", "desc"}},
		{Name: "active", Type: "bool"},
	}

	t.Run("Coerces and validates valid string map", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{
			"limit":  "25",
			"sort":   "desc",
			"active": "true",
		}

		errs := validator.ValidateStringMap(data, fields)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got: %+v", errs)
		}
	})

	t.Run("Catches invalid coerced types and bounds in string map", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{
			"limit":  "200",    // exceeds max 100
			"sort":   "random", // invalid enum
			"active": "invalid",
		}

		errs := validator.ValidateStringMap(data, fields)
		if len(errs) != 3 {
			t.Fatalf("expected 3 errors, got %d: %+v", len(errs), errs)
		}
	})
}
