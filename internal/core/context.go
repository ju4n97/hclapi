package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RequestState represents normalized HTTP request metadata extracted at runtime.
type RequestState struct {
	Method  string            `json:"method"`
	Path    map[string]string `json:"path"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

// StepHandler defines the signature for custom native Go step callbacks.
type StepHandler func(ctx *Context) (any, error)

// StepResult wraps the computed outcome of a previously executed pipeline step.
type StepResult struct {
	Result       any   `json:"result"`
	RowsAffected int64 `json:"rows_affected,omitempty"`
}

// Context encapsulates the runtime execution state for a single HTTP request lifecycle.
type Context struct {
	Request        *RequestState         `json:"request"`
	Steps          map[string]StepResult `json:"steps"`
	Args           map[string]any        `json:"args"`
	TimestampEpoch int64                 `json:"timestamp_epoch"`

	RawRequest *http.Request `json:"-"`
}

type contextConfig struct {
	pathParams  []string
	maxBodySize int64
}

// ContextOption configures optional behavior during Context creation.
type ContextOption func(*contextConfig)

// WithPathParams configures route parameter names to extract from the request.
func WithPathParams(paramNames []string) ContextOption {
	return func(c *contextConfig) {
		c.pathParams = paramNames
	}
}

// WithMaxBodySize sets the maximum allowed request body size in bytes.
func WithMaxBodySize(limit int64) ContextOption {
	return func(c *contextConfig) {
		c.maxBodySize = limit
	}
}

// NewContext parses the HTTP request, enforcing max body size and decoding JSON payloads.
func NewContext(w http.ResponseWriter, r *http.Request, opts ...ContextOption) (*Context, error) {
	var cfg contextConfig
	for _, opt := range opts {
		opt(&cfg)
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

	pathParams := make(map[string]string, len(cfg.pathParams))
	for _, name := range cfg.pathParams {
		pathParams[name] = r.PathValue(name)
	}

	var bodyData any
	if r.Body != nil {
		bodyReader := r.Body
		if cfg.maxBodySize > 0 {
			bodyReader = http.MaxBytesReader(w, r.Body, cfg.maxBodySize)
		}

		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				return nil, err // Preserved for engine status 413 mapping
			}
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &bodyData); err != nil {
				return nil, fmt.Errorf("invalid JSON payload: %w", err)
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return &Context{
		Request: &RequestState{
			Method:  r.Method,
			Path:    pathParams,
			Query:   queryParams,
			Headers: headers,
			Body:    bodyData,
		},
		Steps:          make(map[string]StepResult),
		Args:           make(map[string]any),
		TimestampEpoch: time.Now().Unix(),
		RawRequest:     r,
	}, nil
}

// Context returns the underlying request context, or context.Background() if RawRequest is nil.
func (c *Context) Context() context.Context {
	if c != nil && c.RawRequest != nil {
		return c.RawRequest.Context()
	}
	return context.Background()
}

// WithContext returns a shallow copy of the context with an updated underlying request context.
func (c *Context) WithContext(ctx context.Context) *Context {
	if c == nil {
		return nil
	}
	clone := *c
	if c.RawRequest != nil {
		clone.RawRequest = c.RawRequest.WithContext(ctx)
	}
	return &clone
}
