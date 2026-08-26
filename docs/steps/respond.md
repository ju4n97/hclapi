# Respond step

The `respond` step is the terminal operation that serializes payload data and writes the HTTP status code.

## Syntax

```hcl
respond {
  condition = steps.find_user.rows_affected == 0
  status    = 404
  headers = {
    "X-Resource" = "User"
  }
  body = { error = "User record not found" }
}

respond {
  status = 200
  body   = steps.find_user.result
}
```

## Terminal behavior

When a `respond` block executes:
1. The boolean expression in `condition` is evaluated (if omitted, it evaluates to `true`).
2. If `true`, the status and headers are written, the body is serialized as JSON, and the pipeline halts immediately.
3. Subsequent pipeline steps are not executed.
