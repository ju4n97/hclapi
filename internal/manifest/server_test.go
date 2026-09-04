package manifest_test

import (
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/manifest"
)

func TestServerWithDefaults(t *testing.T) {
	t.Parallel()

	t.Run("Empty server receives all defaults", func(t *testing.T) {
		t.Parallel()

		var s manifest.Server
		res := s.WithDefaults()

		if res.Host != "127.0.0.1" {
			t.Errorf("expected host '127.0.0.1', got %q", res.Host)
		}
		if res.Port != 8080 {
			t.Errorf("expected port 8080, got %d", res.Port)
		}
		if res.ReadTimeout.Duration() != 15*time.Second {
			t.Errorf("expected read timeout 15s, got %v", res.ReadTimeout)
		}
		if res.MaxBodySize.Bytes() != 10*1024*1024 {
			t.Errorf("expected max body size 10MB, got %d", res.MaxBodySize.Bytes())
		}
	})

	t.Run("Custom fields are preserved while missing fields get defaults", func(t *testing.T) {
		t.Parallel()

		s := manifest.Server{
			Host: "0.0.0.0",
			Port: 3000,
		}
		res := s.WithDefaults()

		if res.Host != "0.0.0.0" {
			t.Errorf("expected host '0.0.0.0', got %q", res.Host)
		}
		if res.Port != 3000 {
			t.Errorf("expected port 3000, got %d", res.Port)
		}
		if res.WriteTimeout.Duration() != 15*time.Second {
			t.Errorf("expected write timeout 15s, got %v", res.WriteTimeout)
		}
	})
}
