package xgo

import (
	"fmt"
)

// Execute runs a custom Go step safely, recovering from panics.
func Execute[T any](handler func(T) (any, error), ctx T) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in custom go step: %v", r)
		}
	}()

	return handler(ctx)
}
