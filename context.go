package hclapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RequestState holds structured data extracted from the HTTP request.
type RequestState struct {
	Method  string
	Path    map[string]string
	Query   map[string]string
	Headers map[string]string
	Body    any
}

// StepResult stores the output or metadata produced by an executed step.
type StepResult struct {
	Result any
}

// Context represents the state passed sequentially across a pipeline execution.
type Context struct {
	Request *RequestState
	Steps   map[string]StepResult
	Args    map[string]any

	// rawRequest is the internal raw HTTP request that was used to create the Context.
	rawRequest *http.Request
}

// newContext builds a Context from an incoming HTTP request.
func newContext(req *http.Request, pathParamNames []string) *Context {
	queryParams := map[string]string{}
	for k, v := range req.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	headers := map[string]string{}
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}

	pathParams := map[string]string{}
	for _, name := range pathParamNames {
		pathParams[name] = req.PathValue(name)
	}

	var bodyData any
	if req.Body != nil && req.ContentLength > 0 {
		_ = json.NewDecoder(req.Body).Decode(&bodyData)
	}

	return &Context{
		Request: &RequestState{
			Method:  req.Method,
			Path:    pathParams,
			Query:   queryParams,
			Headers: headers,
			Body:    bodyData,
		},
		Steps:      map[string]StepResult{},
		Args:       map[string]any{},
		rawRequest: req,
	}
}
