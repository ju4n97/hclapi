# 04_redis_caching

Demonstrates a cache-aside architecture using Valkey and PostgreSQL with conditional shortcuts and dynamic TTLs.

## Running

```sh
docker compose up -d
```

## Testing

```sh
# 1. Initial request (Cache MISS -> queries Postgres -> writes to Redis)
curl -i http://localhost:8080/api/v1/products/KB-MECH-01

# 2. Immediate follow-up (Cache HIT -> returned from Redis)
curl -i http://localhost:8080/api/v1/products/KB-MECH-01

# 3. Non-existent SKU (404 Not Found)
curl -i http://localhost:8080/api/v1/products/UNKNOWN-SKU
```
