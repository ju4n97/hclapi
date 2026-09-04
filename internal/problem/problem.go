package problem

import (
	"encoding/json"
	"maps"
	"net/http"
	"strings"
)

// ContentType is the standard RFC 9457 media type for Problem Details JSON payloads.
const ContentType = "application/problem+json"

// DefaultTypePrefix is the standard URN namespace prefix for built-in problem types.
const DefaultTypePrefix = "urn:hclapi:error:"

// TypeURI returns a standard URN identifier for a given error slug.
func TypeURI(slug string) string {
	return DefaultTypePrefix + slug
}

// Slugify converts a title into a lowercase hyphenated slug (e.g. "Not Found" -> "not-found").
func Slugify(title string) string {
	parts := strings.Fields(title)
	for i, part := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(part))
	}
	return strings.Join(parts, "-")
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
	Extensions    map[string]any `json:"-"`
}

// MarshalJSON flattens RFC 9457 extension members into the root JSON object.
func (p Problem) MarshalJSON() ([]byte, error) {
	m := make(map[string]any, 8+len(p.Extensions))
	maps.Copy(m, p.Extensions)
	if p.Type != "" {
		m["type"] = p.Type
	}
	if p.Title != "" {
		m["title"] = p.Title
	}
	if p.Status != 0 {
		m["status"] = p.Status
	}
	if p.Detail != "" {
		m["detail"] = p.Detail
	}
	if p.Instance != "" {
		m["instance"] = p.Instance
	}
	if p.Step != "" {
		m["step"] = p.Step
	}
	if len(p.InvalidParams) > 0 {
		m["invalid_params"] = p.InvalidParams
	}
	return json.Marshal(m)
}

// Error implements the standard Go error interface.
func (p Problem) Error() string {
	if p.Detail != "" {
		return p.Title + ": " + p.Detail
	}
	return p.Title
}

// New constructs a Problem with canonical title and type URI derived from the HTTP status code.
func New(status int, detail ...string) Problem {
	title := http.StatusText(status)
	if title == "" {
		title = "Error"
	}

	p := Problem{
		Status: status,
		Title:  title,
		Type:   TypeURI(Slugify(title)),
	}
	if len(detail) > 0 {
		p.Detail = detail[0]
	}
	return p
}

// InvalidParam represents a single field-level schema validation failure.
type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Handler defines the contract for serializing Problem Details to an HTTP client.
type Handler func(w http.ResponseWriter, r *http.Request, p Problem)

// DefaultHandler serializes a Problem as an application/problem+json response.
func DefaultHandler(w http.ResponseWriter, r *http.Request, p Problem) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(p.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
