package validator_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/validator"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	fields := []core.Field{
		{
			Name:     "title",
			Type:     "string",
			Required: true,
		},
		{
			Name:     "completed",
			Type:     "bool",
			Required: false,
			Default:  false,
		},
		{
			Name:     "role",
			Type:     "string",
			Required: false,
			Default:  "member",
		},
		{
			Name:     "bio",
			Type:     "string",
			Required: false,
		},
		{
			Name:     "limit",
			Type:     "int",
			Required: false,
			Default:  int64(20),
		},
	}

	t.Run("Applies defaults and normalizes missing optional fields to nil", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"title": "Clean architecture task",
		}

		normalized := validator.Normalize(input, fields)

		// Explicit value preserved
		if normalized["title"] != "Clean architecture task" {
			t.Errorf("expected title to be preserved, got %v", normalized["title"])
		}

		// Defaults injected
		if normalized["completed"] != false {
			t.Errorf("expected default completed false, got %v", normalized["completed"])
		}
		if normalized["role"] != "member" {
			t.Errorf("expected default role 'member', got %v", normalized["role"])
		}
		if normalized["limit"] != int64(20) {
			t.Errorf("expected default limit 20, got %v", normalized["limit"])
		}

		// Optional field without default explicitly set to nil
		if normalized["bio"] != nil {
			t.Errorf("expected missing bio to normalize to nil, got %v", normalized["bio"])
		}
	})

	t.Run("Does not overwrite explicitly provided values with defaults", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"title":     "Custom task",
			"completed": true,
			"role":      "admin",
			"limit":     int64(50),
			"bio":       "Gopher developer",
		}

		normalized := validator.Normalize(input, fields)

		if normalized["completed"] != true {
			t.Errorf("expected explicit completed true, got %v", normalized["completed"])
		}
		if normalized["role"] != "admin" {
			t.Errorf("expected explicit role 'admin', got %v", normalized["role"])
		}
		if normalized["limit"] != int64(50) {
			t.Errorf("expected explicit limit 50, got %v", normalized["limit"])
		}
		if normalized["bio"] != "Gopher developer" {
			t.Errorf("expected explicit bio to be preserved, got %v", normalized["bio"])
		}
	})

	t.Run("Preserves extra undeclared fields in input map", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"title":       "Task with extras",
			"custom_meta": "extra_data",
		}

		normalized := validator.Normalize(input, fields)

		if normalized["custom_meta"] != "extra_data" {
			t.Errorf("expected undeclared custom_meta to be preserved, got %v", normalized["custom_meta"])
		}
	})
}

func TestNormalizeStringMap(t *testing.T) {
	t.Parallel()

	fields := []core.Field{
		{Name: "page", Type: "int", Default: 1},
		{Name: "limit", Type: "int", Default: int64(25)},
		{Name: "sort", Type: "string", Default: "desc"},
		{Name: "search", Type: "string"},
	}

	t.Run("Injects stringified defaults in-place for missing or empty keys", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{
			"search": "golang",
			"sort":   "", // Empty string should receive default
		}

		validator.NormalizeStringMap(data, fields)

		if data["page"] != "1" {
			t.Errorf("expected default page '1', got %q", data["page"])
		}
		if data["limit"] != "25" {
			t.Errorf("expected default limit '25', got %q", data["limit"])
		}
		if data["sort"] != "desc" {
			t.Errorf("expected default sort 'desc', got %q", data["sort"])
		}
		if data["search"] != "golang" {
			t.Errorf("expected search 'golang' to be preserved, got %q", data["search"])
		}
	})

	t.Run("Preserves existing non-empty values", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{
			"page":  "5",
			"limit": "100",
			"sort":  "asc",
		}

		validator.NormalizeStringMap(data, fields)

		if data["page"] != "5" || data["limit"] != "100" || data["sort"] != "asc" {
			t.Errorf("expected existing values to be preserved, got: %+v", data)
		}
	})
}
