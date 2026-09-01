package xsql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ju4n97/hclapi/internal/connectors/connsql"
)

var namedParamRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)

// StepResult holds the extracted rows, single row pointer, and affected rows count.
type StepResult struct {
	Rows         []map[string]any
	Row          map[string]any
	RowsAffected int64
}

// RewriteNamedQuery converts @name placeholders into target dialect placeholders
// and produces the ordered argument slice.
func RewriteNamedQuery(query string, args map[string]any, d connsql.Dialect) (string, []any, error) {
	var orderedArgs []any
	var count int

	rewritten := namedParamRegex.ReplaceAllStringFunc(query, func(match string) string {
		paramName := match[1:] // strip '@'
		var val any
		if args != nil {
			val = args[paramName]
		}
		orderedArgs = append(orderedArgs, val)
		placeholder := d.Placeholder(count, paramName)
		count++
		return placeholder
	})

	return rewritten, orderedArgs, nil
}

// ScanRows dynamically scans *sql.Rows into a slice of maps with type normalization and multi-result set support.
func ScanRows(rows *sql.Rows) ([]map[string]any, error) {
	var results []map[string]any

	for {
		cols, err := rows.Columns()
		if err != nil {
			break // No more active result sets
		}

		if len(cols) > 0 {
			for rows.Next() {
				values := make([]any, len(cols))
				valuePtrs := make([]any, len(cols))
				for i := range values {
					valuePtrs[i] = &values[i]
				}

				if err := rows.Scan(valuePtrs...); err != nil {
					return nil, fmt.Errorf("scan column: %w", err)
				}

				rowMap := make(map[string]any, len(cols))
				for i, colName := range cols {
					var val any = values[i]
					switch v := val.(type) {
					case []byte:
						val = string(v)
					case time.Time:
						val = v.Format(time.RFC3339)
					}
					rowMap[colName] = val
				}
				results = append(results, rowMap)
			}
		}

		if !rows.NextResultSet() {
			break
		}
	}

	if results == nil {
		results = []map[string]any{}
	}

	return results, nil
}

// Execute runs a parameterized query or mutation on the connection pool and returns step exports.
func Execute(ctx context.Context, pool *connsql.Pool, query string, args map[string]any) (*StepResult, error) {
	rewrittenQuery, orderedArgs, err := RewriteNamedQuery(query, args, pool.Dialect)
	if err != nil {
		return nil, fmt.Errorf("rewrite query: %w", err)
	}

	trimmedQuery := strings.ToUpper(strings.TrimSpace(rewrittenQuery))
	isRowProducing := strings.HasPrefix(trimmedQuery, "SELECT") ||
		strings.HasPrefix(trimmedQuery, "WITH") ||
		strings.HasPrefix(trimmedQuery, "CALL") ||
		strings.HasPrefix(trimmedQuery, "EXEC") ||
		strings.HasPrefix(trimmedQuery, "PRAGMA") ||
		strings.HasPrefix(trimmedQuery, "SHOW") ||
		strings.HasPrefix(trimmedQuery, "DESC") ||
		strings.HasPrefix(trimmedQuery, "EXPLAIN") ||
		strings.Contains(trimmedQuery, "RETURNING") ||
		strings.Contains(trimmedQuery, "OUTPUT")

	if isRowProducing {
		rows, err := pool.DB.QueryContext(ctx, rewrittenQuery, orderedArgs...)
		if err != nil {
			return nil, fmt.Errorf("query: %w", err)
		}
		defer rows.Close()

		scanned, err := ScanRows(rows)
		if err != nil {
			return nil, err
		}

		var firstRow map[string]any
		if len(scanned) > 0 {
			firstRow = scanned[0]
		}

		return &StepResult{
			Rows:         scanned,
			Row:          firstRow,
			RowsAffected: int64(len(scanned)),
		}, nil
	}

	res, err := pool.DB.ExecContext(ctx, rewrittenQuery, orderedArgs...)
	if err != nil {
		return nil, fmt.Errorf("exec: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}

	return &StepResult{
		Rows:         []map[string]any{},
		Row:          nil,
		RowsAffected: affected,
	}, nil
}
