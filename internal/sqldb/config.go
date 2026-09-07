package sqldb

import (
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"  // MySQL
	_ "github.com/jackc/pgx/v5/stdlib"  // PostgreSQL
	_ "github.com/microsoft/go-mssqldb" // SQL Server
	_ "modernc.org/sqlite"              // SQLite
)

// PoolConfig defines connection pool capacity and lifecycle durations.
type PoolConfig struct {
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
	IdleTimeout time.Duration
}

// DefaultPoolConfig returns baseline production connection pool settings.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpen:     25,
		MaxIdle:     5,
		MaxLifetime: 30 * time.Minute,
		IdleTimeout: 5 * time.Minute,
	}
}

// Config represents the declaration of a single relational database connection pool.
type Config struct {
	Driver string
	Name   string
	Source string
	Pool   PoolConfig
}

// Key returns the short identifier (e.g. "postgres.primary").
func (c Config) Key() string {
	return c.Driver + "." + c.Name
}

// Reference returns the full manifest reference path (e.g. "connection.postgres.primary").
func (c Config) Reference() string {
	return "connection." + c.Driver + "." + c.Name
}

// IsSupportedDriver reports whether the driver identifier belongs to a supported SQL engine.
func IsSupportedDriver(driver string) bool {
	switch strings.ToLower(driver) {
	case "postgres", "cockroachdb", "sqlite", "mysql", "sqlserver", "oracle", "clickhouse", "duckdb":
		return true
	default:
		return false
	}
}

func mapDriverName(driver string) string {
	switch strings.ToLower(driver) {
	case "postgres", "cockroachdb":
		return "pgx"
	default:
		return strings.ToLower(driver)
	}
}
