package connsql

import (
	"errors"
	"strconv"
	"strings"
)

// Dialect provides database-specific SQL parameter placeholder formatting and error code extraction.
type Dialect struct {
	name         string
	placeholder  func(index int, name string) string
	errExtractor func(err error) string
}

// Name returns the canonical name of the dialect.
func (d Dialect) Name() string {
	return d.name
}

// Placeholder returns the positional parameter placeholder for the given parameter index.
func (d Dialect) Placeholder(index int, name string) string {
	if d.placeholder != nil {
		return d.placeholder(index, name)
	}
	return "?"
}

// ExtractErrorCode extracts the standardized driver error code from a database error.
func (d Dialect) ExtractErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if d.errExtractor != nil {
		if code := d.errExtractor(err); code != "" {
			return code
		}
	}
	return defaultExtractSQLErrorCode(err)
}

type pgErrorCoder interface {
	error
	SQLState() string
}

type mySQLError interface {
	error
	Number() uint16
}

type sqliteCoder interface {
	error
	Code() int
}

type mssqlError interface {
	error
	Number() int32
}

type sqlStateCoder interface {
	error
	SQLState() string
}

type codeGetter interface {
	error
	Code() string
}

// Canonical dialect definitions.
var (
	PostgresDialect = Dialect{
		name: "postgres",
		placeholder: func(index int, name string) string {
			return "$" + strconv.Itoa(index+1)
		},
		errExtractor: func(err error) string {
			if e, ok := errors.AsType[pgErrorCoder](err); ok {
				return e.SQLState()
			}
			return ""
		},
	}

	CockroachDialect = Dialect{
		name:         "cockroachdb",
		placeholder:  PostgresDialect.placeholder,
		errExtractor: PostgresDialect.errExtractor,
	}

	MySQLDialect = Dialect{
		name: "mysql",
		placeholder: func(index int, name string) string {
			return "?"
		},
		errExtractor: func(err error) string {
			if e, ok := errors.AsType[mySQLError](err); ok {
				return strconv.FormatUint(uint64(e.Number()), 10)
			}
			return ""
		},
	}

	SQLiteDialect = Dialect{
		name: "sqlite",
		placeholder: func(index int, name string) string {
			return "?"
		},
		errExtractor: func(err error) string {
			if e, ok := errors.AsType[sqliteCoder](err); ok {
				return strconv.Itoa(e.Code())
			}
			return ""
		},
	}

	SQLServerDialect = Dialect{
		name: "sqlserver",
		placeholder: func(index int, name string) string {
			return "@p" + strconv.Itoa(index+1)
		},
		errExtractor: func(err error) string {
			if e, ok := errors.AsType[mssqlError](err); ok {
				return strconv.FormatInt(int64(e.Number()), 10)
			}
			return ""
		},
	}

	OracleDialect = Dialect{
		name: "oracle",
		placeholder: func(index int, name string) string {
			return ":" + strconv.Itoa(index+1)
		},
	}

	ClickHouseDialect = Dialect{
		name: "clickhouse",
		placeholder: func(index int, name string) string {
			return "?"
		},
	}

	DuckDBDialect = Dialect{
		name: "duckdb",
		placeholder: func(index int, name string) string {
			return "?"
		},
	}
)

// ResolveDialect maps a canonical driver name to its corresponding Dialect.
func ResolveDialect(driver string) Dialect {
	switch strings.ToLower(driver) {
	case "postgres":
		return PostgresDialect
	case "cockroachdb":
		return CockroachDialect
	case "mysql":
		return MySQLDialect
	case "sqlite":
		return SQLiteDialect
	case "sqlserver":
		return SQLServerDialect
	case "oracle":
		return OracleDialect
	case "clickhouse":
		return ClickHouseDialect
	case "duckdb":
		return DuckDBDialect
	default:
		return SQLiteDialect
	}
}

func defaultExtractSQLErrorCode(err error) string {
	if coder, ok := errors.AsType[sqlStateCoder](err); ok {
		return coder.SQLState()
	}

	if getter, ok := errors.AsType[codeGetter](err); ok {
		return getter.Code()
	}

	return ""
}
