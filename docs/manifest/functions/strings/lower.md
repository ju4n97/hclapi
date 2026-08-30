---
title: lower
description: Convert all characters in a string to lowercase.
---

# lower

Converts all characters in a string to lowercase.

## Signature

```hcl
lower(str: string) -> string
```

## Examples

```hcl
redis "cache_lookup" {
  connection = connection.redis.cache
  command    = "GET"
  key        = "user:${lower(ctx.request.path.username)}"
}
```
