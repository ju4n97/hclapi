# respond

Terminates the pipeline. Sets the status, headers, and body. No step after
`respond` runs once it fires.

## Declaration

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

## Attributes

| Attribute | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `condition` | `Expression` | `true` | step is skipped if `false` |
| `status` | `int` or `Expression` | `200` | HTTP status code |
| `headers` | `map[string]string` | `{}` | response headers |
| `body` | `any` or `Expression` | previous step's result | payload to serialize |

## Behavior

When `condition` evaluates to `true`, the engine sets
`Content-Type: application/json` unless overridden, applies `status`,
serializes `body`, and terminates the pipeline.

## Examples

Implicit body from the preceding step.

```hcl
sql "find_user" {
  connection = connection.postgres.main
  query      = "SELECT id, name, email FROM users WHERE id = @id"
  args       = { id = ctx.request.path.id }
}

respond {
  status = 200
}
```

204 with an empty body.

```hcl
respond {
  condition = steps.delete_item.rows_affected == 0
  status    = 404
  body      = { error = "Item not found" }
}

respond {
  status = 204
}
```

Custom headers.

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
