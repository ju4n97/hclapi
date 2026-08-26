// Package xrespond serializes payloads and writes final HTTP headers and status codes.
package xrespond

import (
	"encoding/json"
	"net/http"
)

// Write writes the HTTP status code, sets Content-Type to JSON, and serializes
// either the evaluated body or the fallback result from the previous pipeline step.
func Write(w http.ResponseWriter, status int, evaluatedBody any, lastResult any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Explicit body defined in HCL
	if evaluatedBody != nil {
		return json.NewEncoder(w).Encode(evaluatedBody)
	}

	// Implicit fallback to the output of the preceding step
	if lastResult != nil {
		return json.NewEncoder(w).Encode(lastResult)
	}

	return nil
}
