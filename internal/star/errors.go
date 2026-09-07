package star

import (
	"errors"
	"fmt"
)

// ErrMissingExecute is returned when a script does not declare an `execute(ctx)` function.
var ErrMissingExecute = errors.New("script must define an 'execute(ctx)' function")

// CompileError indicates a syntax error or compilation failure in Starlark source code.
type CompileError struct {
	Err error
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("starlark compile error: %v", e.Err)
}

func (e *CompileError) Unwrap() error {
	return e.Err
}

// RuntimeError indicates an unhandled runtime exception, division by zero, or step limit breach.
type RuntimeError struct {
	Err error
}

func (e *RuntimeError) Error() string {
	return fmt.Sprintf("starlark runtime error: %v", e.Err)
}

func (e *RuntimeError) Unwrap() error {
	return e.Err
}
