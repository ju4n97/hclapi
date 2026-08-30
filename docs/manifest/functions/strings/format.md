---
title: format
description: Format a string using standard printf-style verbs.
---

# format

Formats a string using standard `printf` style verbs.

## Signature

```hcl
format(format: string, ...args: any) -> string
```

## Examples

```hcl
respond {
  status = 200
  body = {
    message = format("User %s has %d items in cart", ctx.request.path.name, 4)
  }
}
```
