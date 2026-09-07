package sqldb

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	mssql "github.com/microsoft/go-mssqldb"
	"modernc.org/sqlite"
)

// Dialect provides database-specific placeholder formatting and error code matching.
type Dialect struct {
	name         string
	placeholder  func(index int, name string) string
	errExtractor func(err error) string
}

// Name returns the canonical name of the dialect.
func (d Dialect) Name() string {
	return d.name
}

// Placeholder returns the driver parameter placeholder for the given zero-based index.
func (d Dialect) Placeholder(index int, name string) string {
	if d.placeholder != nil {
		return d.placeholder(index, name)
	}
	return "?"
}

// ExtractErrorCode extracts the driver error code from an error.
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

// MatchErrorCode checks if an actual database error matches a catch block error code.
func (d Dialect) MatchErrorCode(actualCode, targetCode string) bool {
	if actualCode == "" || targetCode == "" {
		return false
	}
	if actualCode == targetCode {
		return true
	}

	// SQLite: Match primary code (e.g. "19") against extended error code (e.g. "2067")
	if d.name == "sqlite" {
		if num, err := strconv.Atoi(actualCode); err == nil {
			if strconv.Itoa(num&0xFF) == targetCode {
				return true
			}
		}
	}

	// PostgreSQL / CockroachDB: Match class prefix (e.g. catch "23" matches "23505")
	if d.name == "postgres" || d.name == "cockroachdb" {
		if strings.HasPrefix(actualCode, targetCode) {
			return true
		}
	}

	return false
}

var (
	PostgresDialect = Dialect{
		name: "postgres",
		placeholder: func(index int, name string) string {
			return "$" + strconv.Itoa(index+1)
		},
		errExtractor: func(err error) string {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return pgErr.Code
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
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				return strconv.FormatUint(uint64(mysqlErr.Number), 10)
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
			var sqliteErr *sqlite.Error
			if errors.As(err, &sqliteErr) {
				return strconv.Itoa(sqliteErr.Code())
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
			var mssqlErr mssql.Error
			if errors.As(err, &mssqlErr) {
				return strconv.FormatInt(int64(mssqlErr.Number), 10)
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

// ResolveDialect returns the canonical Dialect for a given database driver name.
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

type sqlStateCoder interface {
	error
	SQLState() string
}

type codeGetter interface {
	error
	Code() string
}

func defaultExtractSQLErrorCode(err error) string {
	var coder sqlStateCoder
	if errors.As(err, &coder) {
		return coder.SQLState()
	}

	var getter codeGetter
	if errors.As(err, &getter) {
		return getter.Code()
	}

	return ""
}
