package xsql

import (
	"fmt"
	"strings"
)

// Dialect provides database-specific SQL parameter placeholder formatting and error code extraction.
type Dialect interface {
	Name() string
	Placeholder(index int, name string) string
	ExtractErrorCode(err error) string
}

// PostgresDialect formats positional parameters as $1, $2, etc.
type PostgresDialect struct{}

func (PostgresDialect) Name() string { return "postgres" }
func (PostgresDialect) Placeholder(index int, _ string) string {
	return fmt.Sprintf("$%d", index+1)
}
func (PostgresDialect) ExtractErrorCode(err error) string { return extractSQLErrorCode(err) }

// MySQLDialect formats positional parameters as ?.
type MySQLDialect struct{}

func (MySQLDialect) Name() string                       { return "mysql" }
func (MySQLDialect) Placeholder(_ int, _ string) string { return "?" }
func (MySQLDialect) ExtractErrorCode(err error) string  { return extractSQLErrorCode(err) }

// SQLiteDialect formats positional parameters as ?.
type SQLiteDialect struct{}

func (SQLiteDialect) Name() string                       { return "sqlite" }
func (SQLiteDialect) Placeholder(_ int, _ string) string { return "?" }
func (SQLiteDialect) ExtractErrorCode(err error) string  { return extractSQLErrorCode(err) }

// SQLServerDialect formats positional parameters as @p1, @p2, etc.
type SQLServerDialect struct{}

func (SQLServerDialect) Name() string { return "sqlserver" }
func (SQLServerDialect) Placeholder(index int, _ string) string {
	return fmt.Sprintf("@p%d", index+1)
}
func (SQLServerDialect) ExtractErrorCode(err error) string { return extractSQLErrorCode(err) }

// OracleDialect formats positional parameters as :1, :2, etc.
type OracleDialect struct{}

func (OracleDialect) Name() string { return "oracle" }
func (OracleDialect) Placeholder(index int, _ string) string {
	return fmt.Sprintf(":%d", index+1)
}
func (OracleDialect) ExtractErrorCode(err error) string { return extractSQLErrorCode(err) }

// ResolveDialect maps a canonical driver name to its corresponding Dialect.
func ResolveDialect(driver string) Dialect {
	switch strings.ToLower(driver) {
	case "postgres", "cockroachdb":
		return PostgresDialect{}
	case "mysql", "clickhouse", "snowflake", "duckdb":
		return MySQLDialect{}
	case "sqlite":
		return SQLiteDialect{}
	case "sqlserver":
		return SQLServerDialect{}
	case "oracle":
		return OracleDialect{}
	default:
		return SQLiteDialect{}
	}
}

// Helper to extract common SQLState code interfaces if implemented by the driver error.
func extractSQLErrorCode(err error) string {
	if err == nil {
		return ""
	}

	type sqlStateCoder interface {
		SQLState() string
	}
	if coder, ok := err.(sqlStateCoder); ok {
		return coder.SQLState()
	}

	type codeGetter interface {
		Code() string
	}
	if getter, ok := err.(codeGetter); ok {
		return getter.Code()
	}

	return ""
}
