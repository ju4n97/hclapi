package manifest

import (
	"time"

	"github.com/ju4n97/hclapi/internal/scalar"
)

// PoolConfig defines connection pool sizing and lifecycle settings.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime scalar.Duration
	IdleTimeout     scalar.Duration
	Size            int
}

// DefaultPoolConfig returns baseline production connection pool settings.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: scalar.Duration(30 * time.Minute),
		IdleTimeout:     scalar.Duration(5 * time.Minute),
		Size:            20,
	}
}

// Connection represents a resolved connection configuration with driver metadata and pool limits.
type Connection struct {
	Driver string
	Name   string
	URL    string
	Pool   PoolConfig
}

// Key returns the unique identifier for the connection pool (e.g. "postgres.main").
func (c Connection) Key() string {
	return c.Driver + "." + c.Name
}

// Reference returns the full HCL reference path (e.g. "connection.postgres.main").
func (c Connection) Reference() string {
	return "connection." + c.Driver + "." + c.Name
}
