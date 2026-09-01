# clickhouse_analytics

Demonstrates columnar analytical SQL queries and high-throughput telemetry ingestion using ClickHouse.

## Running

Start the database:

```sh
docker compose up -d
```

Then run the server:

```sh
hclapi serve -c ./examples/databases/clickhouse
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/databases/clickhouse
```

## Testing

```sh
# 1. Ingest event
curl -i -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{"user_id": 99, "path": "/dashboard", "duration_ms": 65, "country": "CO"}'

# 2. Get metrics summary
curl -i http://localhost:8080/api/v1/analytics/summary

# 3. Get top countries
curl -i http://localhost:8080/api/v1/analytics/countries
```
