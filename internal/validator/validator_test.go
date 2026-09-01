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

	t.Run("Catches invalid format, bounds, pattern, enum, and duplicates", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"email":    "invalid-email-address", // Bad email format
			"username": "AB!",                   // Bad pattern, length < 3
			"role":     "superadmin",            // Not in enum
			"age":      15,                      // < 18 min
			"tags":     []any{"go", "go"},       // Duplicate items
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

	t.Run("Validates built-in formats", func(t *testing.T) {
		t.Parallel()

		formatFields := []core.Field{
			{Name: "uuid_field", Type: "string", Format: "uuid"},
			{Name: "uri_field", Type: "string", Format: "uri"},
			{Name: "datetime_field", Type: "string", Format: "date-time"},
			{Name: "date_field", Type: "string", Format: "date"},
			{Name: "ipv4_field", Type: "string", Format: "ipv4"},
			{Name: "ipv6_field", Type: "string", Format: "ipv6"},
			{Name: "host_field", Type: "string", Format: "hostname"},
		}

		validData := map[string]any{
			"uuid_field":     "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			"uri_field":      "https://example.com/api",
			"datetime_field": "2026-08-31T20:00:00Z",
			"date_field":     "2026-08-31",
			"ipv4_field":     "192.168.1.1",
			"ipv6_field":     "2001:db8::1",
			"host_field":     "api.example.com",
		}

		if errs := validator.Validate(validData, formatFields); len(errs) != 0 {
			t.Errorf("expected valid formats to pass, got errors: %+v", errs)
		}

		invalidData := map[string]any{
			"uuid_field":     "not-a-uuid",
			"uri_field":      "invalid uri",
			"datetime_field": "31-08-2026",
			"date_field":     "invalid-date",
			"ipv4_field":     "999.999.999.999",
			"ipv6_field":     "192.168.1.1",
			"host_field":     "-bad-host-",
		}

		if errs := validator.Validate(invalidData, formatFields); len(errs) != len(formatFields) {
			t.Errorf("expected %d format errors, got %d: %+v", len(formatFields), len(errs), errs)
		}
	})
}
