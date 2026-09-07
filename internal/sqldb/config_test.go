package sqldb_test

import (
	"testing"

	"github.com/ju4n97/hclapi/internal/sqldb"
)

func TestDialects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		driver      string
		expected    string
		placeholder string
	}{
		{"postgres", "postgres", "$1"},
		{"cockroachdb", "cockroachdb", "$1"},
		{"sqlite", "sqlite", "?"},
		{"mysql", "mysql", "?"},
		{"sqlserver", "sqlserver", "@p1"},
		{"oracle", "oracle", ":1"},
	}

	for _, tt := range tests {
		d := sqldb.ResolveDialect(tt.driver)
		if d.Name() != tt.expected {
			t.Errorf("driver %q: expected dialect %q, got %q", tt.driver, tt.expected, d.Name())
		}
		if p := d.Placeholder(0, "id"); p != tt.placeholder {
			t.Errorf("driver %q: expected placeholder %q, got %q", tt.driver, tt.placeholder, p)
		}
	}
}
