package xgo_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/steps/xgo"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	t.Run("Successful execution", func(t *testing.T) {
		t.Parallel()

		handler := func(ctx *core.Context, args map[string]any) (any, error) {
			name, _ := args["name"].(string)
			return strings.ToUpper(name), nil
		}

		res, err := xgo.Execute(handler, &core.Context{}, map[string]any{"name": "hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "HELLO" {
			t.Errorf("expected 'HELLO', got %v", res)
		}
	})

	t.Run("Standard error propagation", func(t *testing.T) {
		t.Parallel()

		handler := func(ctx *core.Context, args map[string]any) (any, error) {
			return nil, errors.New("db timeout")
		}

		_, err := xgo.Execute(handler, &core.Context{}, nil)
		if err == nil || err.Error() != "db timeout" {
			t.Fatalf("expected 'db timeout', got %v", err)
		}
	})

	t.Run("Panic is safely caught and converted to error", func(t *testing.T) {
		t.Parallel()

		handler := func(ctx *core.Context, args map[string]any) (any, error) {
			panic("nil pointer dereference inside user code")
		}

		_, err := xgo.Execute(handler, &core.Context{}, nil)
		if err == nil {
			t.Fatalf("expected panic to be recovered as error, got nil")
		}
		if !strings.Contains(err.Error(), "panic in custom go step") {
			t.Errorf("expected panic message in error, got: %v", err)
		}
	})
}
