# Example 04: Redis caching

Demonstrates a cache-aside architecture using Redis and PostgreSQL with conditional shortcuts and dynamic TTLs.

## Manifest specification

```hcl
connection "postgres" "db" {
  url = env("DATABASE_URL")
}

connection "redis" "cache" {
  url = env("REDIS_URL")
}

endpoint "GET /api/v1/products/{sku}" {
  request {
    path {
      field "sku" {
        type     = string
        required = true
      }
    }
  }

  pipeline {
    redis "cache_lookup" {
      connection = connection.redis.cache
      command    = "GET"
      key        = "product:{ctx.request.path.sku}"
    }

    # Early return on cache hit
    respond {
      condition = steps.cache_lookup.result != null
      status    = 200
      headers = {
        "X-Cache" = "HIT"
      }
      body = json_decode(steps.cache_lookup.result)
    }

    sql "db_query" {
      connection = connection.postgres.db
      query      = "SELECT * FROM products WHERE sku = @sku"
      args = {
        sku = ctx.request.path.sku
      }
    }

    respond {
      condition = steps.db_query.rows_affected == 0
      status    = 404
      body      = { error = "Product not found" }
    }

    redis "cache_write" {
      connection = connection.redis.cache
      command    = "SET"
      key        = "product:{ctx.request.path.sku}"
      value      = json_encode(steps.db_query.result)
      ttl        = "30m"
    }

    respond {
      status  = 200
      headers = {
        "X-Cache" = "MISS"
      }
      body = steps.db_query.result
    }
  }
}
```
