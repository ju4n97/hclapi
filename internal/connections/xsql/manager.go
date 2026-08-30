package xsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ju4n97/hclapi/internal/core"
)

// Pool wraps an active *sql.DB pool with its database dialect and configuration metadata.
type Pool struct {
	DB      *sql.DB
	Dialect Dialect
	Config  core.Connection
}

// Manager manages the lifecycle, pooling, and retrieval of active database connections.
type Manager struct {
	mu    sync.Mutex
	pools map[string]*Pool
}

// NewManager initializes an empty connection pool manager.
func NewManager() *Manager {
	return &Manager{
		pools: make(map[string]*Pool),
	}
}

// IsSupportedDriver checks if the driver identifier belongs to a supported SQL engine.
func IsSupportedDriver(driver string) bool {
	switch strings.ToLower(driver) {
	case "postgres",
		"sqlite",
		"mysql",
		"sqlserver",
		"oracle",
		"cockroachdb",
		"clickhouse",
		"snowflake",
		"duckdb":
		return true
	default:
		return false
	}
}

// mapDriverName maps user manifest driver labels to registered database/sql driver names.
func mapDriverName(driver string) string {
	switch strings.ToLower(driver) {
	case "postgres", "cockroachdb":
		return "pgx"
	default:
		return strings.ToLower(driver)
	}
}

// Open initializes a connection pool, applies pool sizing limits, pings the database, and registers it.
func (m *Manager) Open(ctx context.Context, conn core.Connection) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := conn.Key()
	if _, exists := m.pools[key]; exists {
		return fmt.Errorf("connection %q is already registered", conn.Reference())
	}

	driverName := mapDriverName(conn.Driver)
	db, err := sql.Open(driverName, conn.URL)
	if err != nil {
		return fmt.Errorf("failed to open database handle for %q: %w", conn.Reference(), err)
	}

	db.SetMaxOpenConns(conn.Pool.MaxOpenConns)
	db.SetMaxIdleConns(conn.Pool.MaxIdleConns)
	db.SetConnMaxLifetime(conn.Pool.ConnMaxLifetime.Duration())
	db.SetConnMaxIdleTime(conn.Pool.IdleTimeout.Duration())

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect to database %q: %w", conn.Reference(), err)
	}

	pool := &Pool{
		DB:      db,
		Dialect: ResolveDialect(driverName),
		Config:  conn,
	}

	m.pools[key] = pool
	return nil
}

// Get retrieves an active connection pool by key ("postgres.main") or reference ("connection.postgres.main").
func (m *Manager) Get(keyOrRef string) (*Pool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanKey := strings.TrimPrefix(keyOrRef, "connection.")
	pool, exists := m.pools[cleanKey]
	return pool, exists
}

// Close gracefully closes all active database connection pools.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, pool := range m.pools {
		if err := pool.DB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database pool %q: %w", pool.Config.Reference(), err))
		}
	}

	m.pools = make(map[string]*Pool)
	return errors.Join(errs...)
}
