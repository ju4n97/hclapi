package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ju4n97/hclapi/internal/manifest"
)

// StepResult represents arbitrary step-specific outputs.
type StepResult = map[string]any

// RequestState represents normalized HTTP request metadata extracted at runtime.
type RequestState struct {
	Method  string            `json:"method"`
	Path    map[string]string `json:"path"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

// PathParam returns the route path parameter value, or fallback if absent.
func (r *RequestState) PathParam(key string, fallback ...string) string {
	if r != nil && r.Path != nil {
		if val, ok := r.Path[key]; ok && val != "" {
			return val
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// QueryParam returns the query parameter value, or fallback if absent.
func (r *RequestState) QueryParam(key string, fallback ...string) string {
	if r != nil && r.Query != nil {
		if val, ok := r.Query[key]; ok {
			return val
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// Header returns the header value for key (case-insensitive).
func (r *RequestState) Header(key string) string {
	if r == nil || r.Headers == nil {
		return ""
	}
	return r.Headers[strings.ToLower(key)]
}

// ExecutionContext encapsulates the runtime state for a single HTTP request pipeline execution.
type ExecutionContext struct {
	Request        *RequestState         `json:"request"`
	Steps          map[string]StepResult `json:"steps"`
	TimestampEpoch int64                 `json:"timestamp_epoch"`
	IngressTime    time.Time             `json:"-"`
	Server         manifest.Server       `json:"-"`
	RawRequest     *http.Request         `json:"-"`

	mu sync.RWMutex
}

// Step encapsulates the invocation state and arguments for a single Go step.
type Step struct {
	*ExecutionContext
	Name string `json:"name"`
	Args Args   `json:"args"`
}

// StepHandler defines the signature for custom native Go step callbacks.
type StepHandler func(ctx context.Context, step *Step) (any, error)

type executionContextConfig struct {
	pathParams []string
	server     manifest.Server
}

// ExecutionContextOption configures optional behavior during context creation.
type ExecutionContextOption func(*executionContextConfig)

// WithPathParams configures route parameter names to extract from the request.
func WithPathParams(paramNames []string) ExecutionContextOption {
	return func(c *executionContextConfig) { c.pathParams = paramNames }
}

// WithServer attaches the resolved server configuration to the execution context.
func WithServer(server manifest.Server) ExecutionContextOption {
	return func(c *executionContextConfig) { c.server = server }
}

// NewExecutionContext parses the incoming HTTP request, enforces body limits, and initializes state.
func NewExecutionContext(w http.ResponseWriter, r *http.Request, opts ...ExecutionContextOption) (*ExecutionContext, error) {
	var cfg executionContextConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	ingressTime := time.Now().UTC()

	// Extract path parameters
	pathParams := make(map[string]string, len(cfg.pathParams))
	for _, name := range cfg.pathParams {
		pathParams[name] = r.PathValue(name)
	}

	// Extract query parameters
	queryParams := make(map[string]string, len(r.URL.Query()))
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	// Extract lowercased headers
	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}

	// Ingest and validate request body
	var bodyData any
	if r.Body != nil && r.Body != http.NoBody {
		bodyReader := r.Body
		maxBodySize := cfg.server.MaxBodySize.Bytes()
		if maxBodySize > 0 && w != nil {
			bodyReader = http.MaxBytesReader(w, r.Body, maxBodySize)
		}

		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
				return nil, fmt.Errorf("request body too large: %w", maxBytesErr)
			}
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

		// Restore r.Body with the consumed bytes so subsequent readers don't fail
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		if len(bodyBytes) > 0 {
			contentType := strings.ToLower(r.Header.Get("Content-Type"))

			if contentType != "" && !strings.Contains(contentType, "json") {
				bodyData = string(bodyBytes)
			} else {
				if err := json.Unmarshal(bodyBytes, &bodyData); err != nil {
					return nil, fmt.Errorf("invalid JSON payload: %w", err)
				}
			}
		}
	} else {
		r.Body = http.NoBody
	}

	return &ExecutionContext{
		Request: &RequestState{
			Method:  r.Method,
			Path:    pathParams,
			Query:   queryParams,
			Headers: headers,
			Body:    bodyData,
		},
		Steps:          make(map[string]StepResult),
		TimestampEpoch: ingressTime.Unix(),
		IngressTime:    ingressTime,
		Server:         cfg.server.WithDefaults(),
		RawRequest:     r,
	}, nil
}

// NewStep creates an isolated, thread-safe Step invocation referencing this ExecutionContext.
func (e *ExecutionContext) NewStep(name string, args Args) *Step {
	return &Step{
		ExecutionContext: e,
		Name:             name,
		Args:             args,
	}
}

// SetStepResult safely records the output of a step (concurrent-safe for parallel execution).
func (e *ExecutionContext) SetStepResult(stepName string, result StepResult) {
	if e == nil || stepName == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Steps[stepName] = result
}

// GetStepResult safely retrieves prior step output (concurrent-safe).
func (e *ExecutionContext) GetStepResult(stepName string) (StepResult, bool) {
	if e == nil {
		return nil, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	res, ok := e.Steps[stepName]
	return res, ok
}

// SnapshotSteps returns a shallow copy of all step results recorded so far.
// Use this when passing steps to the HCL evaluation engine to prevent concurrent map read/write panics.
func (e *ExecutionContext) SnapshotSteps() map[string]StepResult {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	snapshot := make(map[string]StepResult, len(e.Steps))
	maps.Copy(snapshot, e.Steps)
	return snapshot
}

// Context returns the underlying standard Go request context.
func (e *ExecutionContext) Context() context.Context {
	if e != nil && e.RawRequest != nil {
		return e.RawRequest.Context()
	}
	return context.Background()
}

// WithContext returns a shallow copy of ExecutionContext with an updated standard library context.
func (e *ExecutionContext) WithContext(ctx context.Context) *ExecutionContext {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	var rawReq *http.Request
	if e.RawRequest != nil {
		rawReq = e.RawRequest.WithContext(ctx)
	}

	return &ExecutionContext{
		Request:        e.Request,
		Steps:          e.Steps,
		TimestampEpoch: e.TimestampEpoch,
		IngressTime:    e.IngressTime,
		Server:         e.Server,
		RawRequest:     rawReq,
	}
}
