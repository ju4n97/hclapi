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

// StepResult represents arbitrary step-specific outputs.
type StepResult = map[string]any

// StepHandler defines the signature for custom native Go step callbacks.
type StepHandler func(ctx *Context, args map[string]any) (any, error)

// Context encapsulates the runtime execution state for a single HTTP request lifecycle.
type Context struct {
	Request        *RequestState             `json:"request"`
	Steps          map[string]map[string]any `json:"steps"`
	TimestampEpoch int64                     `json:"timestamp_epoch"`
	IngressTime    time.Time                 `json:"-"`
	RawRequest     *http.Request             `json:"-"`
}

type contextConfig struct {
	pathParams  []string
	maxBodySize int64
}

// ContextOption configures optional behavior during Context creation.
type ContextOption func(*contextConfig)

// WithPathParams configures route parameter names to extract from the request.
func WithPathParams(paramNames []string) ContextOption {
	return func(c *contextConfig) { c.pathParams = paramNames }
}

// WithMaxBodySize sets the maximum allowed request body size in bytes.
func WithMaxBodySize(limit int64) ContextOption {
	return func(c *contextConfig) { c.maxBodySize = limit }
}

// NewContext parses the HTTP request, enforcing max body size and decoding payloads.
func NewContext(w http.ResponseWriter, r *http.Request, opts ...ContextOption) (*Context, error) {
	var cfg contextConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	ingressTime := time.Now().UTC()

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
	if r.Body != nil && r.Body != http.NoBody {
		bodyReader := r.Body
		if cfg.maxBodySize > 0 && w != nil {
			bodyReader = http.MaxBytesReader(w, r.Body, cfg.maxBodySize)
		}

		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
				return nil, fmt.Errorf("request body too large: %w", maxBytesErr)
			}
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

		if len(bodyBytes) > 0 {
			contentType := strings.ToLower(r.Header.Get("Content-Type"))

			// If Content-Type is explicitly non-JSON (e.g. text/plain, form-urlencoded, xml)
			if contentType != "" && !strings.Contains(contentType, "json") {
				bodyData = string(bodyBytes)
			} else {
				// Default to JSON decoding for application/json, *+json, or omitted Content-Type
				if err := json.Unmarshal(bodyBytes, &bodyData); err != nil {
					return nil, fmt.Errorf("invalid JSON payload: %w", err)
				}
			}

			// Restore r.Body so custom Go steps can re-read raw payload from ctx.RawRequest.Body
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
		TimestampEpoch: ingressTime.Unix(),
		IngressTime:    ingressTime,
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

// WithContext returns a shallow copy of Context with an updated underlying request context.
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
