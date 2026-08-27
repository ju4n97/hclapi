package core_test

import (
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/core"
)

func TestDuration(t *testing.T) {
	t.Parallel()

	t.Run("UnmarshalText with valid strings", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			input    string
			expected time.Duration
		}{
			{"10s", 10 * time.Second},
			{"500ms", 500 * time.Millisecond},
			{"15m", 15 * time.Minute},
			{"1h", 1 * time.Hour},
			{"1h30m", 90 * time.Minute},
			{"", 0},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				t.Parallel()

				var d core.Duration
				err := d.UnmarshalText([]byte(tt.input))
				if err != nil {
					t.Fatalf("unexpected error for %q: %v", tt.input, err)
				}

				if d.Duration() != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, d.Duration())
				}
			})
		}
	})

	t.Run("UnmarshalText with invalid strings returns error", func(t *testing.T) {
		t.Parallel()

		invalidInputs := []string{
			"invalid",
			"10x",
			"100years",
			"abc",
		}

		for _, input := range invalidInputs {
			t.Run(input, func(t *testing.T) {
				t.Parallel()

				var d core.Duration
				err := d.UnmarshalText([]byte(input))
				if err == nil {
					t.Fatalf("expected error for invalid input %q, got nil", input)
				}
			})
		}
	})

	t.Run("MarshalText and String formatting", func(t *testing.T) {
		t.Parallel()

		d := core.Duration(15 * time.Minute)

		if d.String() != "15m0s" {
			t.Errorf("expected '15m0s', got %q", d.String())
		}

		b, err := d.MarshalText()
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}

		if string(b) != "15m0s" {
			t.Errorf("expected '15m0s', got %q", string(b))
		}
	})
}
