package scalar_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/scalar"
)

func TestParseByteSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input       string
		expected    scalar.ByteSize
		expectError bool
	}{
		{"10MB", 10 * 1000 * 1000, false},
		{"10MiB", 10 * 1024 * 1024, false},
		{"512B", 512, false},
		{"1GB", 1000 * 1000 * 1000, false},
		{"1GiB", 1024 * 1024 * 1024, false},
		{"1024", 1024, false},
		{"", 0, false},
		{"invalid", 0, true},
		{"10XB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			res, err := scalar.ParseByteSize(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tt.expected {
				t.Errorf("expected %d bytes, got %d", tt.expected, res)
			}
		})
	}
}
