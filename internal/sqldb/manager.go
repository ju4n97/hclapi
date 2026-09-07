package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Manager manages the thread-safe lifecycle and retrieval of active database connection pools.
type Manager struct {
	mu    sync.RWMutex
	pools map[string]*Pool
}

// NewManager initializes an empty connection pool manager.
func NewManager() *Manager {
	return &Manager{
		pools: make(map[string]*Pool),
	}
}

// Open initializes a connection pool, applies tuning bounds, pings with a timeout, and registers it.
func (m *Manager) Open(ctx context.Context, cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := cfg.Key()
	if _, exists := m.pools[key]; exists {
		return fmt.Errorf("connection %q is already registered", cfg.Reference())
	}

	driverName := mapDriverName(cfg.Driver)
	db, err := sql.Open(driverName, cfg.Source)
	if err != nil {
		return fmt.Errorf("open %s driver: %w", driverName, err)
	}

	// Apply connection pool limits
	db.SetMaxOpenConns(cfg.Pool.MaxOpen)
	db.SetMaxIdleConns(cfg.Pool.MaxIdle)
	db.SetConnMaxLifetime(cfg.Pool.MaxLifetime)
	db.SetConnMaxIdleTime(cfg.Pool.IdleTimeout)

	// Verify connectivity with a 5s timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return fmt.Errorf("ping database: %w", err)
	}

	pool := &Pool{
		DB:      db,
		Dialect: ResolveDialect(cfg.Driver),
		Config:  cfg,
	}

	m.pools[key] = pool
	return nil
}

// Get retrieves an active pool by short key ("postgres.main") or full reference ("connection.postgres.main").
func (m *Manager) Get(keyOrRef string) (*Pool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleanKey := strings.TrimPrefix(keyOrRef, "connection.")
	pool, exists := m.pools[cleanKey]
	return pool, exists
}

// Close gracefully closes all active database pools.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, pool := range m.pools {
		if err := pool.DB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close pool %q: %w", pool.Config.Reference(), err))
		}
	}

	m.pools = make(map[string]*Pool)
	return errors.Join(errs...)
}
