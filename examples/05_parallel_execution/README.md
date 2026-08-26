# 05_parallel_execution

Demonstrates concurrent DAG execution running three isolated SQL queries in parallel to aggregate user dashboard metrics into a single response.

## Running

```sh
docker compose up -d
```

## Testing

```sh
# Fetch aggregate dashboard overview
curl -i http://localhost:8080/api/v1/accounts/1/overview
```
