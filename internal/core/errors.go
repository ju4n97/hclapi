package core

import (
	"encoding/json"
	"net/http"
)

// InvalidParam represents a single field validation failure.
type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ProblemDetailsError represents an RFC 9457 compliant error object.
type ProblemDetailsError struct {
	Type          string         `json:"type,omitempty"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	Step          string         `json:"step,omitempty"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
	Extensions    map[string]any `json:"extensions,omitempty"`
}

// Error implements the standard error interface.
func (p ProblemDetailsError) Error() string {
	if p.Detail != "" {
		return p.Title + ": " + p.Detail
	}
	return p.Title
}

// ErrorHandler defines the contract for customizing API error serialization.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, problem ProblemDetailsError)

// DefaultErrorHandler returns a ProblemDetails with default values.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, problem ProblemDetailsError) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
