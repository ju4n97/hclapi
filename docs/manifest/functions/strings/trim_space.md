---
title: trim_space
description: Remove leading and trailing whitespace from a string.
---

# trim_space

Removes leading and trailing whitespace characters (spaces, tabs, newlines) from a string.

## Signature

```hcl
trim_space(str: string) -> string
```

## Examples

```hcl
sql "search" {
  connection = connection.postgres.main
  query      = "SELECT id, name FROM users WHERE name ILIKE @query"
  args       = { query = "%${trim_space(ctx.request.query.q)}%" }
}
```
