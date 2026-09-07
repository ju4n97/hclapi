package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// QueryResult holds extracted rows, single row pointer, and affected rows count.
type QueryResult struct {
	Rows         []map[string]any
	Row          map[string]any
	RowsAffected int64
}

// Pool wraps an active database connection pool with dialect handling.
type Pool struct {
	DB      *sql.DB
	Dialect Dialect
	Config  Config
}

// Execute runs a parameterized SQL query or mutation on the pool and returns normalized rows.
func (p *Pool) Execute(ctx context.Context, query string, args map[string]any) (*QueryResult, error) {
	rewrittenQuery, orderedArgs, err := RewriteNamedQuery(query, args, p.Dialect)
	if err != nil {
		return nil, fmt.Errorf("rewrite query: %w", err)
	}

	trimmed := strings.ToUpper(strings.TrimSpace(rewrittenQuery))
	isRowProducing := strings.HasPrefix(trimmed, "SELECT") ||
		strings.HasPrefix(trimmed, "WITH") ||
		strings.HasPrefix(trimmed, "CALL") ||
		strings.HasPrefix(trimmed, "EXEC") ||
		strings.HasPrefix(trimmed, "PRAGMA") ||
		strings.HasPrefix(trimmed, "SHOW") ||
		strings.HasPrefix(trimmed, "DESC") ||
		strings.HasPrefix(trimmed, "EXPLAIN") ||
		strings.Contains(trimmed, "RETURNING") ||
		strings.Contains(trimmed, "OUTPUT")

	if isRowProducing {
		rows, err := p.DB.QueryContext(ctx, rewrittenQuery, orderedArgs...)
		if err != nil {
			return nil, fmt.Errorf("query: %w", err)
		}
		defer rows.Close()

		scanned, err := scanRows(rows)
		if err != nil {
			return nil, err
		}

		var firstRow map[string]any
		if len(scanned) > 0 {
			firstRow = scanned[0]
		}

		return &QueryResult{
			Rows:         scanned,
			Row:          firstRow,
			RowsAffected: int64(len(scanned)),
		}, nil
	}

	res, err := p.DB.ExecContext(ctx, rewrittenQuery, orderedArgs...)
	if err != nil {
		return nil, fmt.Errorf("exec: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}

	return &QueryResult{
		Rows:         []map[string]any{},
		Row:          nil,
		RowsAffected: affected,
	}, nil
}

// scanRows scans *sql.Rows dynamically into a slice of maps with type normalization.
func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	var results []map[string]any

	for {
		cols, err := rows.Columns()
		if err != nil {
			break
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
					val := values[i]
					switch v := val.(type) {
					case []byte:
						val = string(v) // Normalize raw bytes to string
					case time.Time:
						val = v.Format(time.RFC3339) // Normalize timestamps to RFC 3339
					}
					rowMap[colName] = val
				}
				results = append(results, rowMap)
			}
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("read rows: %w", err)
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
