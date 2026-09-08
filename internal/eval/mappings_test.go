package eval_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/runtime"
)

func TestAnyToCty_StructsAndSlices(t *testing.T) {
	t.Parallel()

	type Item struct {
		URL      string `json:"url"`
		FreshURL string `json:"fresh_url,omitempty"`
		Type     string `json:"type"`
		Internal string `json:"-"` // Must be ignored
	}

	t.Run("converts struct to cty object with json tags", func(t *testing.T) {
		t.Parallel()

		item := Item{
			URL:      "http://localhost:8080/asset?id=1",
			FreshURL: "https://storage.googleapis.com/asset1",
			Type:     "image",
			Internal: "secret",
		}

		execCtx := &runtime.ExecutionContext{
			Steps: map[string]runtime.StepResult{
				"test_step": {
					"result": item,
				},
			},
		}

		res, err := eval.Any(parseExpr(t, `steps.test_step.result`), execCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T (%v)", res, res)
		}

		if m["url"] != "http://localhost:8080/asset?id=1" {
			t.Errorf("url mismatch: %v", m["url"])
		}
		if m["fresh_url"] != "https://storage.googleapis.com/asset1" {
			t.Errorf("fresh_url mismatch: %v", m["fresh_url"])
		}
		if m["type"] != "image" {
			t.Errorf("type mismatch: %v", m["type"])
		}
		if _, exists := m["Internal"]; exists {
			t.Errorf("field with json:\"-\" should be omitted")
		}
	})

	t.Run("converts slice of structs to cty tuple of objects", func(t *testing.T) {
		t.Parallel()

		items := []Item{
			{URL: "http://asset/1", FreshURL: "https://gcs/1", Type: "image"},
			{URL: "http://asset/2", FreshURL: "https://gcs/2", Type: "viewer"},
		}

		execCtx := &runtime.ExecutionContext{
			Steps: map[string]runtime.StepResult{
				"batch": {
					"items": items,
				},
			},
		}

		res, err := eval.Any(parseExpr(t, `steps.batch.items`), execCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		slice, ok := res.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T (%v)", res, res)
		}

		if len(slice) != 2 {
			t.Fatalf("expected 2 items, got %d", len(slice))
		}

		first, ok := slice[0].(map[string]any)
		if !ok {
			t.Fatalf("expected slice elements to be map[string]any, got %T", slice[0])
		}
		if first["url"] != "http://asset/1" {
			t.Errorf("expected first url 'http://asset/1', got %v", first["url"])
		}
	})

	t.Run("converts pointer to struct and handles nil pointer", func(t *testing.T) {
		t.Parallel()

		ptrItem := &Item{URL: "http://ptr", Type: "report"}
		var nilPtr *Item

		execCtx := &runtime.ExecutionContext{
			Steps: map[string]runtime.StepResult{
				"ptrs": {
					"valid": ptrItem,
					"nil":   nilPtr,
				},
			},
		}

		resValid, err := eval.Any(parseExpr(t, `steps.ptrs.valid`), execCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := resValid.(map[string]any)
		if !ok || m["url"] != "http://ptr" {
			t.Errorf("expected valid pointer conversion: %+v", resValid)
		}

		resNil, err := eval.Any(parseExpr(t, `steps.ptrs.nil`), execCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resNil != nil {
			t.Errorf("expected nil pointer to evaluate to nil, got %v", resNil)
		}
	})
}
