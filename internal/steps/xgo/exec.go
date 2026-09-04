package xgo

import (
	"context"
	"fmt"

	"github.com/ju4n97/hclapi/internal/runtime"
)

// Execute runs a custom Go step safely, recovering from panics.
func Execute(ctx context.Context, handler runtime.StepHandler, step *runtime.Step) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			stepName := "anonymous"
			if step != nil && step.Name != "" {
				stepName = step.Name
			}
			err = fmt.Errorf("panic in custom go step %q: %v", stepName, r)
		}
	}()

	return handler(ctx, step)
}
