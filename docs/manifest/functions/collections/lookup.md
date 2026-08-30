---
title: lookup
description: Retrieve a map value with a fallback default.
---

# lookup

Retrieves the value of a single key from a map, returning a fallback default if the key does not exist.

## Signature

```hcl
lookup(map: map, key: string, default: any) -> any
```

## Examples

```hcl
respond {
  status = 200
  body = {
    role = lookup(ctx.request.body, "role", "standard_user")
  }
}
```
