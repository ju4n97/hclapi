---
title: coalesce
description: Return the first argument that is not null and not an empty string.
---

# coalesce

Evaluates arguments sequentially and returns the first argument that is not `null` and not an empty string `""`.

## Signature

```hcl
coalesce(...values: any) -> any
```

## Parameters

| Parameter   | Type  | Required   | Description                  |
| :---------- | :---- | :--------- | :--------------------------- |
| `...values` | `any` | at least 1 | Candidate values to evaluate |

## Return value

Returns the first non-null/non-empty value.

## Examples

```hcl
sql "list_items" {
  connection = connection.postgres.main
  query      = "SELECT id, name FROM items LIMIT @limit OFFSET @offset"
  args = {
    limit  = coalesce(ctx.request.query.limit, 20)
    offset = coalesce(ctx.request.query.offset, 0)
  }
}
```
