package problem

import (
	"encoding/json"
	"net/http"
)

// ContentType is the standard RFC 9457 media type for Problem Details JSON payloads.
const ContentType = "application/problem+json"

// DefaultTypePrefix is the standard URN namespace prefix for built-in error types.
const DefaultTypePrefix = "urn:hclapi:error:"

// TypeURI returns a standard URN identifier for a given error slug.
func TypeURI(slug string) string {
	return DefaultTypePrefix + slug
}

// Problem represents an RFC 9457 compliant error object.
type Problem struct {
	Type          string         `json:"type,omitempty"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	Step          string         `json:"step,omitempty"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
	Extensions    map[string]any `json:"extensions,omitempty"`
}

// Error implements the standard Go error interface.
func (p Problem) Error() string {
	if p.Detail != "" {
		return p.Title + ": " + p.Detail
	}
	return p.Title
}

// InvalidParam represents a single field-level schema validation failure.
type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Handler defines the contract for serializing Problem Details to an HTTP client.
type Handler func(w http.ResponseWriter, r *http.Request, p Problem)

// DefaultHandler serializes Details as an application/problem+json response.
func DefaultHandler(w http.ResponseWriter, r *http.Request, p Problem) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(p.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
