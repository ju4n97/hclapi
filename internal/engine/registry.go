package engine

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ju4n97/hclapi/internal/runtime"
)

// StepRegistry provides thread-safe registration and resolution of native Go step handlers.
type StepRegistry struct {
	mu       sync.RWMutex
	handlers map[string]runtime.StepHandler
}

// NewStepRegistry initializes an empty step registry.
func NewStepRegistry() *StepRegistry {
	return &StepRegistry{
		handlers: make(map[string]runtime.StepHandler),
	}
}

// Register stores a named callback in the registry. It fails if the name is already in use.
func (r *StepRegistry) Register(name string, handler runtime.StepHandler) error {
	if name == "" {
		return errors.New("step name cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("step %q: handler cannot be nil", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("step %q already registered", name)
	}

	r.handlers[name] = handler
	return nil
}

// Get safely retrieves a registered callback by name.
func (r *StepRegistry) Get(name string) (runtime.StepHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.handlers[name]
	return h, ok
}
