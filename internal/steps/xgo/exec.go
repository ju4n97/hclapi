package xgo

import (
	"fmt"

	"github.com/ju4n97/hclapi/internal/core"
)

// Execute runs a custom Go step safely, recovering from panics.
func Execute(handler core.StepHandler, ctx *core.Context, args map[string]any) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in custom go step: %v", r)
		}
	}()

	return handler(ctx, args)
}
