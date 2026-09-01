# duckdb_analytics

Demonstrates embedded, in-process columnar SQL queries with zero Docker or external server dependencies using DuckDB.

## Running

```sh
hclapi serve -c ./examples/databases/duckdb
```

or

```sh
hclapi serve -c ./examples/databases/duckdb
```

## Testing

```sh
# 1. Ingest telemetry events
curl -i -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "path": "/checkout", "duration_ms": 110, "country": "US"}'

curl -i -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{"user_id": 2, "path": "/checkout", "duration_ms": 190, "country": "CO"}'

# 2. Query in-memory aggregations
curl -i http://localhost:8080/api/v1/analytics/summary
```
