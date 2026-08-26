package hclapi

import "net/http"

// Context is the shared state passed through an execution pipeline.
type Context struct {
	Request *http.Request
	Args    map[string]any // Arguments evaluated from the HCL `args = {}` block
	// TODO: add auth claims and step results
}
