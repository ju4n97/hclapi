// Package runtime manages request-scoped execution state, step outputs,
// and context lifecycles for pipeline runs.
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

	"github.com/ju4n97/hclapi/internal/problem"
)

// StepResult represents the exported outputs of a completed pipeline step.
type StepResult = map[string]any

// RequestState holds normalized, read-only metadata extracted from an incoming HTTP request.
type RequestState struct {
	Method  string            `json:"method"`
	Path    map[string]string `json:"path"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

// PathParam returns the route path parameter for key, or fallback if absent.
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

// QueryParam returns the URL query parameter for key, or fallback if absent.
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

// Header returns the case-insensitive HTTP request header for key.
func (r *RequestState) Header(key string) string {
	if r == nil || r.Headers == nil {
		return ""
	}
	return r.Headers[strings.ToLower(key)]
}

// ExecutionContext encapsulates the state, context, and accumulated step outputs for a single request.
type ExecutionContext struct {
	Request           *RequestState         `json:"request"`
	Steps             map[string]StepResult `json:"steps"`
	TimestampEpoch    int64                 `json:"timestamp_epoch"`
	IngressTime       time.Time             `json:"-"`
	MaxBodySize       int64                 `json:"-"`
	ProblemTypePrefix string                `json:"-"`
	RawRequest        *http.Request         `json:"-"`

	mu sync.RWMutex
}

// Step wraps ExecutionContext with step-specific metadata and evaluated arguments for a Go step handler.
type Step struct {
	*ExecutionContext
	Name string `json:"name"`
	Args Args   `json:"args"`
}

// Problem constructs an RFC 9457 Problem Details error bound to this step's name and request instance.
func (s *Step) Problem(status int, detail string) problem.Problem {
	p := problem.New(status, detail)
	p.Step = s.Name
	if s.RawRequest != nil && s.RawRequest.URL != nil {
		p.Instance = s.RawRequest.URL.Path
	}
	if s.ProblemTypePrefix != "" {
		slug := problem.Slugify(p.Title)
		if strings.HasPrefix(s.ProblemTypePrefix, "http://") || strings.HasPrefix(s.ProblemTypePrefix, "https://") {
			p.Type = strings.TrimSuffix(s.ProblemTypePrefix, "/") + "/" + slug
		} else {
			p.Type = s.ProblemTypePrefix + slug
		}
	}
	return p
}

// StepHandler defines the signature for custom native Go step callbacks.
type StepHandler func(ctx context.Context, step *Step) (any, error)

type executionContextConfig struct {
	pathParams        []string
	maxBodySize       int64
	problemTypePrefix string
}

// ExecutionContextOption configures optional behavior when initializing an ExecutionContext.
type ExecutionContextOption func(*executionContextConfig)

// WithPathParams configures the route path parameter names to extract from the request.
func WithPathParams(paramNames []string) ExecutionContextOption {
	return func(c *executionContextConfig) { c.pathParams = paramNames }
}

// WithMaxBodySize enforces an upper byte limit on the incoming HTTP request body.
func WithMaxBodySize(maxBytes int64) ExecutionContextOption {
	return func(c *executionContextConfig) { c.maxBodySize = maxBytes }
}

// WithProblemTypePrefix sets the base URI prefix used when deriving RFC 9457 problem error types.
func WithProblemTypePrefix(prefix string) ExecutionContextOption {
	return func(c *executionContextConfig) { c.problemTypePrefix = prefix }
}

// NewExecutionContext parses the incoming HTTP request, applies size constraints, and initializes pipeline state.
func NewExecutionContext(w http.ResponseWriter, r *http.Request, opts ...ExecutionContextOption) (*ExecutionContext, error) {
	var cfg executionContextConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	ingressTime := time.Now().UTC()

	pathParams := make(map[string]string, len(cfg.pathParams))
	for _, name := range cfg.pathParams {
		pathParams[name] = r.PathValue(name)
	}

	queryParams := make(map[string]string, len(r.URL.Query()))
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}

	var bodyData any
	if r.Body != nil && r.Body != http.NoBody {
		bodyReader := r.Body
		if cfg.maxBodySize > 0 && w != nil {
			bodyReader = http.MaxBytesReader(w, r.Body, cfg.maxBodySize)
		}

		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				return nil, fmt.Errorf("request body too large: %w", maxBytesErr)
			}
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

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
		Steps:             make(map[string]StepResult),
		TimestampEpoch:    ingressTime.Unix(),
		IngressTime:       ingressTime,
		MaxBodySize:       cfg.maxBodySize,
		ProblemTypePrefix: cfg.problemTypePrefix,
		RawRequest:        r,
	}, nil
}

// NewStep creates an execution handle for a named step with evaluated arguments.
func (e *ExecutionContext) NewStep(name string, args Args) *Step {
	return &Step{
		ExecutionContext: e,
		Name:             name,
		Args:             args,
	}
}

// SetStepResult stores the exported outputs of a completed step.
func (e *ExecutionContext) SetStepResult(stepName string, result StepResult) {
	if e == nil || stepName == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Steps[stepName] = result
}

// GetStepResult retrieves the outputs of a previously executed step.
func (e *ExecutionContext) GetStepResult(stepName string) (StepResult, bool) {
	if e == nil {
		return nil, false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	res, ok := e.Steps[stepName]
	return res, ok
}

// SnapshotSteps returns a shallow copy of all step outputs recorded up to this point.
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

// Context returns the underlying standard library request context.
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
		Request:           e.Request,
		Steps:             e.Steps,
		TimestampEpoch:    e.TimestampEpoch,
		IngressTime:       e.IngressTime,
		MaxBodySize:       e.MaxBodySize,
		ProblemTypePrefix: e.ProblemTypePrefix,
		RawRequest:        rawReq,
	}
}
