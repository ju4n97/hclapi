package problem_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ju4n97/hclapi/internal/problem"
)

func TestProblem_New(t *testing.T) {
	t.Parallel()

	t.Run("canonical title and type derivation", func(t *testing.T) {
		t.Parallel()

		p := problem.New(http.StatusUnauthorized, "Token expired")

		if p.Status != 401 {
			t.Errorf("Status = %d; want 401", p.Status)
		}
		if p.Title != "Unauthorized" {
			t.Errorf("Title = %q; want 'Unauthorized'", p.Title)
		}
		if p.Type != "urn:hclapi:error:unauthorized" {
			t.Errorf("Type = %q; want 'urn:hclapi:error:unauthorized'", p.Type)
		}
		if p.Detail != "Token expired" {
			t.Errorf("Detail = %q; want 'Token expired'", p.Detail)
		}
	})

	t.Run("error interface implementation", func(t *testing.T) {
		t.Parallel()

		p := problem.New(http.StatusNotFound, "Resource missing")
		if got := p.Error(); got != "Not Found: Resource missing" {
			t.Errorf("Error() = %q; want 'Not Found: Resource missing'", got)
		}

		pNoDetail := problem.New(http.StatusForbidden)
		if got := pNoDetail.Error(); got != "Forbidden" {
			t.Errorf("Error() = %q; want 'Forbidden'", got)
		}
	})
}

func TestProblem_Slugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Not Found", "not-found"},
		{"Bad Request", "bad-request"},
		{"Payload Too Large", "payload-too-large"},
		{"  Multiple   Spaces  ", "multiple-spaces"},
		{"Already-Slugified", "already-slugified"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := problem.Slugify(tt.input); got != tt.want {
				t.Errorf("Slugify(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProblem_ExtensionsRootSerialization_RFC9457(t *testing.T) {
	t.Parallel()

	p := problem.Problem{
		Status:   http.StatusTooManyRequests,
		Title:    "Rate Limit Exceeded",
		Detail:   "Quota exhausted for current window.",
		Type:     "urn:hclapi:error:rate-limit-exceeded",
		Instance: "/api/v1/orders",
		Extensions: map[string]any{
			"retry_after_seconds": 60,
			"tier":                "hobby",
			"reset_epoch":         1771968000,
		},
	}

	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(bytes, &rawMap); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	// Verify extensions appear directly at the root level of the JSON payload
	if rawMap["retry_after_seconds"] != float64(60) {
		t.Errorf("expected root 'retry_after_seconds' = 60, got %v", rawMap["retry_after_seconds"])
	}
	if rawMap["tier"] != "hobby" {
		t.Errorf("expected root 'tier' = 'hobby', got %v", rawMap["tier"])
	}
	if rawMap["reset_epoch"] != float64(1771968000) {
		t.Errorf("expected root 'reset_epoch' = 1771968000, got %v", rawMap["reset_epoch"])
	}

	// Ensure there is no nested "extensions" wrapper object
	if _, exists := rawMap["extensions"]; exists {
		t.Errorf("found unwanted nested 'extensions' key in JSON payload: %+v", rawMap)
	}
}
