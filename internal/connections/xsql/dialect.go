package xsql

import (
	"strconv"
	"strings"
)

// Dialect provides database-specific SQL parameter placeholder formatting and error code extraction.
type Dialect struct {
	name        string
	placeholder func(index int, name string) string
}

// Name returns the name of the dialect.
func (d Dialect) Name() string {
	return d.name
}

// Placeholder returns the placeholder for the given parameter index and name.
func (d Dialect) Placeholder(index int, name string) string {
	if d.placeholder != nil {
		return d.placeholder(index, name)
	}
	return "?"
}

// ExtractErrorCode extracts the underlying SQL error code from the given error.
func (d Dialect) ExtractErrorCode(err error) string {
	return extractSQLErrorCode(err)
}

var (
	PostgresDialect = Dialect{
		name: "postgres",
		placeholder: func(index int, name string) string {
			return "$" + strconv.Itoa(index+1)
		},
	}
	MySQLDialect = Dialect{
		name: "mysql",
		placeholder: func(index int, name string) string {
			return "?"
		},
	}
	SQLiteDialect = Dialect{
		name: "sqlite",
		placeholder: func(index int, name string) string {
			return "?"
		},
	}
	SQLServerDialect = Dialect{
		name: "sqlserver",
		placeholder: func(index int, name string) string {
			return "@p" + strconv.Itoa(index+1)
		},
	}
	OracleDialect = Dialect{
		name: "oracle",
		placeholder: func(index int, name string) string {
			return ":" + strconv.Itoa(index+1)
		},
	}
)

// ResolveDialect maps a canonical driver name to its corresponding Dialect.
func ResolveDialect(driver string) Dialect {
	switch strings.ToLower(driver) {
	case "postgres", "cockroachdb":
		return PostgresDialect
	case "mysql", "clickhouse", "snowflake", "duckdb":
		return MySQLDialect
	case "sqlite":
		return SQLiteDialect
	case "sqlserver":
		return SQLServerDialect
	case "oracle":
		return OracleDialect
	default:
		return SQLiteDialect
	}
}

func extractSQLErrorCode(err error) string {
	if err == nil {
		return ""
	}

	type sqlStateCoder interface{ SQLState() string }
	if coder, ok := err.(sqlStateCoder); ok {
		return coder.SQLState()
	}

	type codeGetter interface{ Code() string }
	if getter, ok := err.(codeGetter); ok {
		return getter.Code()
	}

	return ""
}
