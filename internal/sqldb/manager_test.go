package sqldb_test

import (
	"testing"
	"time"

	"github.com/ju4n97/hclapi/internal/sqldb"
)

func TestManager(t *testing.T) {
	t.Parallel()

	t.Run("Registers and retrieves pool by short key and full reference", func(t *testing.T) {
		t.Parallel()

		mgr := sqldb.NewManager()
		cfg := sqldb.Config{
			Driver: "sqlite",
			Name:   "main",
			Source: "file::memory:?cache=shared",
			Pool: sqldb.PoolConfig{
				MaxOpen:     10,
				MaxIdle:     2,
				MaxLifetime: 15 * time.Minute,
				IdleTimeout: 5 * time.Minute,
			},
		}

		if err := mgr.Open(t.Context(), cfg); err != nil {
			t.Fatalf("unexpected open error: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })

		// Short lookup
		pool, ok := mgr.Get("sqlite.main")
		if !ok || pool == nil {
			t.Fatal("expected pool for sqlite.main")
		}

		// Full reference lookup
		poolRef, okRef := mgr.Get("connection.sqlite.main")
		if !okRef || poolRef == nil {
			t.Fatal("expected pool for connection.sqlite.main")
		}

		// Duplicate registration fails
		if err := mgr.Open(t.Context(), cfg); err == nil {
			t.Fatal("expected error on duplicate pool registration, got nil")
		}
	})

	t.Run("Fails fast on unreachable database DSN", func(t *testing.T) {
		t.Parallel()

		mgr := sqldb.NewManager()
		cfg := sqldb.Config{
			Driver: "sqlite",
			Name:   "bad",
			Source: "file:/non_existent_folder_99999/db.sqlite?mode=ro",
			Pool:   sqldb.DefaultPoolConfig(),
		}

		if err := mgr.Open(t.Context(), cfg); err == nil {
			t.Fatal("expected ping failure, got nil")
		}
	})
}
