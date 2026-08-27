---
title: Respond
description: HTTP response serialization, conditional headers, and status code delivery.
---

# Respond

The `respond` step is the terminal handler of a pipeline. It formats the HTTP status code, sets custom response headers, serializes the response payload, and flushes the output to the client.

## Block declaration

```hcl
respond {
  condition = steps.find_user.rows_affected == 0
  status    = 404
  headers = {
    "X-Custom-Header" = "Value"
  }
  body = {
    error = "Resource not found"
  }
}
```

## Attribute reference

| Attribute | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| **`condition`** | `Expression` | `true` | Boolean condition expression. If false, the step is skipped. |
| **`status`** | `int` / `Expression` | `200` | HTTP response status code to return to the client. |
| **`headers`** | `map[string]string` | `{}` | Custom HTTP response headers. |
| **`body`** | `any` / `Expression` | `lastResult` | Payload to serialize. If omitted, falls back to preceding step output. |

## Terminal behavior

When a `respond` step evaluates its `condition` to `true` (or when an unconditional `respond` is reached), the engine:

1. Sets `Content-Type: application/json` (unless overridden in `headers`).
2. Applies the declared HTTP status code.
3. Serializes the `body` into JSON.
4. **Terminates the pipeline immediately**, preventing downstream steps from running.

If `condition` evaluates to `false`, the step is skipped and execution proceeds to the next step.

## Usage patterns

### 1. Implicit payload fallback

When `body` is omitted, the `respond` step automatically serializes the result of the immediately preceding step:

```hcl
endpoint "GET /api/v1/users/{id}" {
  pipeline {
    sql "find_user" {
      connection = connection.postgres.main
      query      = "SELECT id, name, email FROM users WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    # Automatically serializes steps.find_user.result
    respond {
      status = 200
    }
  }
}
```

### 2. No content response (HTTP 204)

When handling deletion operations, omitting `body` with status `204` flushes an empty response:

```hcl
endpoint "DELETE /api/v1/items/{id}" {
  pipeline {
    sql "delete_item" {
      connection = connection.postgres.main
      query      = "DELETE FROM items WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    respond {
      condition = steps.delete_item.rows_affected == 0
      status    = 404
      body      = { error = "Item not found" }
    }

    # Returns HTTP 204 No Content with empty body
    respond {
      status = 204
    }
  }
}
```

### 3. Custom caching headers

```hcl
respond {
  status = 200
  headers = {
    "Cache-Control" = "public, max-age=3600"
    "X-Rate-Limit"  = "1000"
  }
  body = steps.calculate_stats.result
}
```

### 4. Dynamic error payloads

```hcl
respond {
  condition = steps.verify_token.result.is_valid == false
  status    = 401
  body = {
    type     = "https://hclapi.dev/errors/unauthorized"
    title    = "Unauthorized"
    status   = 401
    detail   = "The provided authentication token has expired or is invalid."
    instance = ctx.request.path
  }
}
```
