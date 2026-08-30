---
title: min
description: Return the smallest value from a sequence of numbers.
---

# min

Returns the smallest value from a sequence of numbers.

## Signature

```hcl
min(...numbers: number) -> number
```

## Examples

```hcl
sql "bounded_query" {
  connection = connection.postgres.main
  query      = "SELECT id FROM logs LIMIT @limit"
  args = {
    # Cap requested page size at 100 max
    limit = min(coalesce(ctx.request.query.limit, 20), 100)
  }
}
```
