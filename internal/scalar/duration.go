package scalar

import (
	"fmt"
	"time"
)

// Duration wraps a time.Duration with universal text deserialization.
type Duration time.Duration

// Duration returns the duration as a time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// String returns the duration as a human-readable string.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (d *Duration) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "" {
		*d = Duration(0)
		return nil
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}

	*d = Duration(parsed)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}
