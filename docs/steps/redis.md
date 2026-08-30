---
title: redis
description: Execute Redis or Valkey key-value commands from a pipeline.
---

# redis

Executes key-value commands against a Redis or Valkey connection pool. Supports cache reads, writes with TTL, key deletions, and counters.

## Declaration

```hcl
redis "cache_lookup" {
  connection = connection.redis.cache
  command    = "GET"
  key        = "product:${ctx.request.path.sku}"
}
```

## Attributes

| Attribute    | Type         | Required  | Description                                          |
| :----------- | :----------- | :-------- | :--------------------------------------------------- |
| label        | `string`     | yes       | Step identifier; output is written to `steps.<name>` |
| `connection` | `connection` | yes       | Redis connection pool reference                      |
| `command`    | `string`     | yes       | `GET`, `SET`, `DEL`, `INCR`, or `EXISTS`             |
| `key`        | `string`     | yes       | Cache key; supports dynamic string interpolation     |
| `value`      | `any`        | for `SET` | Value to store (typically `json_encode(...)`)        |
| `ttl`        | `Duration`   | no        | Expiration duration for `SET` commands               |

## Exported outputs

| Field                | Type  | Description                                                                                                     |
| :------------------- | :---- | :-------------------------------------------------------------------------------------------------------------- |
| `steps.<name>.value` | `any` | The retrieved cache value (or `null` on a cache miss), `"OK"` for `SET`, count for `DEL`, or integer for `INCR` |

## Examples

### Cache-aside pattern

```hcl
endpoint "GET /api/v1/products/{sku}" {
  pipeline {
    redis "cache_lookup" {
      connection = connection.redis.cache
      command    = "GET"
      key        = "cache:product:${ctx.request.path.sku}"
    }

    respond {
      condition = steps.cache_lookup.value != null
      status    = 200
      headers   = { "X-Cache" = "HIT" }
      body      = json_decode(steps.cache_lookup.value)
    }

    sql "db_query" {
      connection = connection.postgres.main
      query      = "SELECT id, sku, name, price FROM products WHERE sku = @sku"
      args       = { sku = ctx.request.path.sku }
    }

    respond {
      condition = steps.db_query.rows_affected == 0
      status    = 404
      body      = { error = "Product not found" }
    }

    redis "cache_write" {
      connection = connection.redis.cache
      command    = "SET"
      key        = "cache:product:${ctx.request.path.sku}"
      value      = json_encode(steps.db_query.row)
      ttl        = "30m"
    }

    respond {
      status  = 200
      headers = { "X-Cache" = "MISS" }
      body    = steps.db_query.row
    }
  }
}
```
