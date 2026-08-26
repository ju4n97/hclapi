# Redis step

The `redis` step executes commands against a Redis connection pool.

## Cache-aside workflow

```mermaid
flowchart TD
    Req([Request]) --> Lookup[redis: GET user:{id}]
    Lookup --> Check{Hit in Redis?}
    Check -->|Yes| FastReturn[respond 200: X-Cache: HIT]
    Check -->|No| DBQuery[sql: SELECT FROM users]
    DBQuery --> CacheWrite[redis: SET user:{id} with TTL]
    CacheWrite --> MissReturn[respond 200: X-Cache: MISS]
```

## Syntax

```hcl
pipeline {
  redis "cache_get" {
    connection = connection.redis.cache
    command    = "GET"
    key        = "item:{ctx.request.path.id}"
  }

  # Fast-path early return on cache hit
  respond {
    condition = steps.cache_get.result != null
    status    = 200
    headers   = { "X-Cache" = "HIT" }
    body      = json_decode(steps.cache_get.result)
  }

  # Fallback to database
  sql "db_get" {
    connection = connection.postgres.main
    query      = "SELECT * FROM items WHERE id = @id"
    args       = { id = ctx.request.path.id }
  }

  redis "cache_set" {
    connection = connection.redis.cache
    command    = "SET"
    key        = "item:{ctx.request.path.id}"
    value      = json_encode(steps.db_get.result)
    ttl        = "1h"
  }

  respond {
    status  = 200
    headers = { "X-Cache" = "MISS" }
    body    = steps.db_get.result
  }
}
```
