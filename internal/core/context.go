package core

import (
	"bytes"
	"encoding/json"
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
	Result any `json:"result"`
}

// Context encapsulates the runtime execution state for a single HTTP request lifecycle.
type Context struct {
	Request        *RequestState         `json:"request"`
	Steps          map[string]StepResult `json:"steps"`
	Args           map[string]any        `json:"args"`
	TimestampEpoch int64                 `json:"timestamp_epoch"`
	RawRequest     *http.Request         `json:"-"`
}

// NewContext parses an incoming HTTP request into an isolated pipeline Context.
func NewContext(req *http.Request, pathParamNames []string) *Context {
	queryParams := make(map[string]string, len(req.URL.Query()))
	for k, v := range req.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	headers := make(map[string]string, len(req.Header))
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}

	pathParams := make(map[string]string, len(pathParamNames))
	for _, name := range pathParamNames {
		pathParams[name] = req.PathValue(name)
	}

	var bodyData any
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil && len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &bodyData)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return &Context{
		Request: &RequestState{
			Method:  req.Method,
			Path:    pathParams,
			Query:   queryParams,
			Headers: headers,
			Body:    bodyData,
		},
		Steps:          make(map[string]StepResult),
		Args:           make(map[string]any),
		TimestampEpoch: time.Now().Unix(),
		RawRequest:     req,
	}
}
