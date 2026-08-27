---
title: Redis
description: Key-value cache-aside lookups, storage, and dynamic TTL management.
---

# Redis

The `redis` step executes key-value caching operations against Redis or Valkey connection pools. It supports cache-aside reads, writes with dynamic TTL expiration, key deletions, and atomic counters.

## Block declaration

```hcl
redis "<name>" {
  connection = connection.redis.<name>
  command    = "GET"
  key        = "cache:product:${ctx.request.path.sku}"
}
```

## Attribute reference

| Attribute | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| **`name`** (Label) | `string` | Yes | Unique step identifier used under `steps.<name>.result`. |
| **`connection`** | `connection` | Yes | Reference to a declared Redis connection block. |
| **`command`** | `string` | Yes | Command to execute (`"GET"`, `"SET"`, `"DEL"`, `"INCR"`, `"EXISTS"`). |
| **`key`** | `string` | Yes | Cache key. Supports `${...}` string interpolation. |
| **`value`** | `any` | For `SET` | Value to store. Typically serialized via `json_encode()`. |
| **`ttl`** | `Duration` | No | Expiration duration for `SET` commands (e.g. `"15m"`, `"24h"`). |

## Supported commands

| Command | Required attributes | Output type (`steps.<name>.result`) | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `key` | `string` or `null` | Retrieves the value at the key. Returns `null` on cache miss. |
| `SET` | `key`, `value`, `ttl` (optional) | `string` (`"OK"`) | Stores the value at the key with an optional expiration TTL. |
| `DEL` | `key` | `int` | Deletes the specified key. Returns the number of keys removed. |
| `INCR` | `key` | `int` | Increments the integer value of the key by one. |
| `EXISTS` | `key` | `bool` | Checks if the key exists in the database. |

## The cache-aside pattern

The standard cache-aside pattern is implemented by chaining a cache lookup step, an early `respond` on cache hit, a database fallback query on miss, and a subsequent cache write step:

```hcl
endpoint "GET /api/v1/products/{sku}" {
  description = "Fetches product metadata with Redis cache-aside fallback"

  pipeline {
    # 1. Probe the cache
    redis "cache_lookup" {
      connection = connection.redis.cache
      command    = "GET"
      key        = "cache:product:${ctx.request.path.sku}"
    }

    # 2. Fast-path: Return cached payload immediately on cache hit
    respond {
      condition = steps.cache_lookup.result != null
      status    = 200
      headers = {
        "X-Cache"       = "HIT"
        "Cache-Control" = "public, max-age=1800"
      }
      body = json_decode(steps.cache_lookup.result)
    }

    # 3. Cache miss: Query primary database
    sql "db_query" {
      connection = connection.postgres.main
      query      = <<-SQL
        SELECT id, sku, name, price_cents, inventory
        FROM products
        WHERE sku = @sku
      SQL
      args = {
        sku = ctx.request.path.sku
      }
    }

    # 4. Handle 404 if product does not exist in database
    respond {
      condition = steps.db_query.rows_affected == 0
      status    = 404
      body      = { error = "Product not found" }
    }

    # 5. Populate cache with fresh database record (30 minute TTL)
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
```
