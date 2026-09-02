package core

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ByteSize represents a quantity of bytes that can be unmarshaled from text.
type ByteSize int64

// ByteSize constants represent byte sizes.
const (
	B   ByteSize = 1
	KB  ByteSize = 1000 * B
	KiB ByteSize = 1024 * B
	MB  ByteSize = 1000 * KB
	MiB ByteSize = 1024 * KiB
	GB  ByteSize = 1000 * MB
	GiB ByteSize = 1024 * MiB
	TB  ByteSize = 1000 * GB
	TiB ByteSize = 1024 * GiB
)

// Bytes returns the byte size as an integer.
func (b ByteSize) Bytes() int64 {
	return int64(b)
}

// String returns the byte size as a human-readable string.
func (b ByteSize) String() string {
	switch {
	case b >= GiB && b%GiB == 0:
		return fmt.Sprintf("%dGiB", b/GiB)
	case b >= GB && b%GB == 0:
		return fmt.Sprintf("%dGB", b/GB)
	case b >= MiB && b%MiB == 0:
		return fmt.Sprintf("%dMiB", b/MiB)
	case b >= MB && b%MB == 0:
		return fmt.Sprintf("%dMB", b/MB)
	case b >= KiB && b%KiB == 0:
		return fmt.Sprintf("%dKiB", b/KiB)
	case b >= KB && b%KB == 0:
		return fmt.Sprintf("%dKB", b/KB)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// ParseByteSize parses human-readable byte strings into a ByteSize.
func ParseByteSize(s string) (ByteSize, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Split numeric part and unit part
	i := 0
	for i < len(s) && (unicode.IsDigit(rune(s[i])) || s[i] == '.') {
		i++
	}

	numStr := strings.TrimSpace(s[:i])
	unitStr := strings.ToUpper(strings.TrimSpace(s[i:]))

	if numStr == "" {
		return 0, fmt.Errorf("invalid byte size %q: missing numeric value", s)
	}

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size %q: %w", s, err)
	}

	var multiplier ByteSize
	switch unitStr {
	case "", "B":
		multiplier = B
	case "K", "KB":
		multiplier = KB
	case "KIB":
		multiplier = KiB
	case "M", "MB":
		multiplier = MB
	case "MIB":
		multiplier = MiB
	case "G", "GB":
		multiplier = GB
	case "GIB":
		multiplier = GiB
	case "T", "TB":
		multiplier = TB
	case "TIB":
		multiplier = TiB
	default:
		return 0, fmt.Errorf("invalid byte size %q: unknown unit %q", s, unitStr)
	}

	return ByteSize(val * float64(multiplier)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *ByteSize) UnmarshalText(text []byte) error {
	parsed, err := ParseByteSize(string(text))
	if err != nil {
		return err
	}

	*b = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (b ByteSize) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}
