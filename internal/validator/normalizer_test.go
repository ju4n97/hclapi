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
		{
			Name:     "score",
			Type:     "float",
			Required: false,
		},
	}

	t.Run("Injects defaults and normalizes missing optional fields to nil", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"title": "Only title provided",
		}

		normalized := validator.Normalize(input, fields)

		if normalized["title"] != "Only title provided" {
			t.Errorf("expected title to be preserved, got %v", normalized["title"])
		}
		if normalized["completed"] != false {
			t.Errorf("expected default completed false, got %v", normalized["completed"])
		}
		if normalized["limit"] != int64(20) {
			t.Errorf("expected default limit 20, got %v", normalized["limit"])
		}
		if normalized["bio"] != nil {
			t.Errorf("expected missing bio without default to normalize to nil, got %v", normalized["bio"])
		}
	})

	t.Run("Coerces string inputs from query and path parameters", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"title":     "Typed strings",
			"limit":     "50",
			"score":     "98.5",
			"completed": "true",
		}

		normalized := validator.Normalize(input, fields)

		if normalized["limit"] != int64(50) {
			t.Errorf("expected coerced int64 50, got %T (%v)", normalized["limit"], normalized["limit"])
		}
		if normalized["score"] != 98.5 {
			t.Errorf("expected coerced float64 98.5, got %T (%v)", normalized["score"], normalized["score"])
		}
		if normalized["completed"] != true {
			t.Errorf("expected coerced bool true, got %T (%v)", normalized["completed"], normalized["completed"])
		}
	})
}
