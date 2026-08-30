---
title: split
description: Split a string into a list of substrings separated by separator.
---

# split

Splits a string into a list of substrings separated by `separator`.

## Signature

```hcl
split(separator: string, str: string) -> list(string)
```

## Examples

```hcl
respond {
  status = 200
  body = {
    filters = split(",", coalesce(ctx.request.query.filters, ""))
  }
}
```
