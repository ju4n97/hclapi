---
title: Dynamic expression evaluation
description: Syntax, operators, and functions for runtime HCL expressions in Hclapi pipelines.
---

# Dynamic expression evaluation

HCL attributes in Hclapi pipelines support dynamic expressions evaluated at request time. Expressions enable parameter mapping, conditional routing, string interpolation, and response body rendering using data from the execution context (`ctx`).

## Expression evaluation points

Expressions are evaluated at specific block attributes across the pipeline lifecycle:

```hcl
endpoint "GET /api/v1/products/{sku}" {
  pipeline {
    # 1. String interpolation in step parameters
    redis "cache_lookup" {
      connection = connection.redis.main
      command    = "GET"
      key        = "cache:product:${ctx.request.path.sku}"
    }

    # 2. Boolean conditions for early return
    respond {
      condition = steps.cache_lookup.result != null
      status    = 200
      headers = {
        "X-Cache" = "HIT"
      }
      body = json_decode(steps.cache_lookup.result)
    }

    # 3. Dynamic argument mapping from request context
    sql "fetch_product" {
      connection = connection.postgres.main
      query      = <<-SQL
        SELECT id, sku, name, price_cents, stock
        FROM products
        WHERE sku = @sku
      SQL
      args = {
        sku = ctx.request.path.sku
      }
    }

    # 4. Error branching on query execution metrics
    respond {
      condition = steps.fetch_product.rows_affected == 0
      status    = 404
      body = {
        error = "Product not found"
        sku   = ctx.request.path.sku
      }
    }

    # 5. Serialization of upstream step results
    respond {
      status = 200
      headers = {
        "X-Cache" = "MISS"
      }
      body = steps.fetch_product.result
    }
  }
}
```

## Comparison and logical operators

Conditional attributes like `condition` evaluate expressions down to a boolean `true` or `false`.

| Operator | Type | Example |
| :--- | :--- | :--- |
| `==`, `!=` | Equality | `steps.lookup.rows_affected == 0` |
| `>`, `>=` | Numeric comparison | `ctx.request.body.score >= 100` |
| `<`, `<=` | Numeric comparison | `steps.inventory.result.count < 5` |
| `&&` | Logical AND | `steps.auth.result.valid == true && ctx.request.body.admin == true` |
| `\|\|` | Logical OR | `ctx.request.query.format == "csv" \|\| ctx.request.query.format == "xlsx"` |
| `!` | Logical NOT | `!steps.user.result.is_active` |

### Null checks

State fields that may be absent or uninitialized evaluate safely against `null`:

```hcl
respond {
  # Evaluates to true if the cache step returned a populated value
  condition = steps.cache_lookup.result != null
  status    = 200
  body      = steps.cache_lookup.result
}
```

## String interpolation

Strings declared with `${...}` syntax interpolate values from `ctx.request` or `steps`:

```hcl
redis "session_write" {
  connection = connection.redis.sessions
  command    = "SET"
  key        = "session:${ctx.request.headers.x_session_id}:user"
  value      = steps.find_user.result.id
  ttl        = "15m"
}
```

Multi-variable interpolation is supported within string literals:

```hcl
respond {
  status = 200
  body = {
    message = "User ${ctx.request.body.name} registered under organization ${steps.org.result.slug}"
  }
}
```

## Argument mapping in steps

The `args` attribute maps execution context values to parameterized statement inputs:

```hcl
sql "update_account" {
  connection = connection.postgres.main
  query      = <<-SQL
    UPDATE accounts
    SET name = @name, updated_at = NOW()
    WHERE id = @id
    RETURNING id, name, updated_at
  SQL
  args = {
    id   = ctx.request.path.id
    name = steps.sanitize_input.result.clean_name
  }
}
```

Parameters declared with `@<param>` in SQL blocks are automatically sanitized and bound to prevent SQL injection vulnerabilities.

## Inline object and list construction

Expressions can define complex JSON responses directly in HCL:

```hcl
respond {
  status = 201
  body = {
    account = steps.create_account.result
    metadata = {
      requested_by = ctx.request.headers.authorization
      created_at   = ctx.timestamp_epoch
      tags         = ["api", "v1", ctx.request.query.environment]
    }
  }
}
```

## Built-in functions

Hclapi provides built-in expression functions for environment reading and data serialization:

### `env`

Reads an environment variable from the host operating system. Commonly used in connection pool configurations and authorization definitions:

```hcl
connection "postgres" "primary" {
  url = env("DATABASE_URL")
}
```

### `json_encode`

Converts a step output or structured object into a serialized JSON string:

```hcl
redis "cache_product" {
  connection = connection.redis.cache
  command    = "SET"
  key        = "product:${ctx.request.path.id}"
  value      = json_encode(steps.fetch_product.result)
  ttl        = "30m"
}
```

### `json_decode`

Parses a JSON string retrieved from an external data source or cache into an object for output serialization:

```hcl
respond {
  condition = steps.cache_lookup.result != null
  status    = 200
  body      = json_decode(steps.cache_lookup.result)
}
```
