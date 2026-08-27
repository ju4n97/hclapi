server {
  host = "127.0.0.1"
  port = 8080
}

connection "postgres" "db" {
  url = env("DATABASE_URL")
}

connection "redis" "cache" {
  url = env("REDIS_URL")

  pool {
    size         = 20
    idle_timeout = "5m"
  }
}

endpoint "GET /api/v1/products/{sku}" {
  description = "Fetches product metadata with Redis cache-aside fallback."

  request {
    path {
      field "sku" {
        type     = string
        required = true
      }
    }
  }

  pipeline {
    # 1. Probe cache
    redis "cache_lookup" {
      connection = connection.redis.cache
      command    = "GET"
      key        = "cache:product:${ctx.request.path.sku}"
    }

    # 2. Fast-path return on cache hit
    respond {
      condition = steps.cache_lookup.result != null
      status    = 200
      headers = {
        "X-Cache"       = "HIT"
        "Cache-Control" = "public, max-age=1800"
      }
      body = json_decode(steps.cache_lookup.result)
    }

    # 3. Cache miss: query database
    sql "db_query" {
      connection = connection.postgres.db
      query      = <<-SQL
        SELECT id, sku, name, price_cents, inventory_count, updated_at
        FROM products
        WHERE sku = @sku
      SQL
      args = {
        sku = ctx.request.path.sku
      }
    }

    # 4. Handle not found
    respond {
      condition = steps.db_query.rows_affected == 0
      status    = 404
      body      = { error = "Product SKU not found" }
    }

    # 5. Populate cache with a 30-minute TTL
    redis "cache_write" {
      connection = connection.redis.cache
      command    = "SET"
      key        = "cache:product:${ctx.request.path.sku}"
      value      = json_encode(steps.db_query.result)
      ttl        = "30m"
    }

    # 6. Return fresh response
    respond {
      status = 200
      headers = {
        "X-Cache"       = "MISS"
        "Cache-Control" = "public, max-age=1800"
      }
      body = steps.db_query.result
    }
  }
}