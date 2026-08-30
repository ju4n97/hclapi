---
title: length
description: Return the number of elements in a list, keys in a map, or characters in a string.
---

# length

Returns the total number of elements in a list, keys in a map, or characters in a string.

## Signature

```hcl
length(collection: any) -> int
```

## Examples

```hcl
respond {
  condition = length(steps.fetch_orders.result) == 0
  status    = 204
}

respond {
  status = 200
  body = {
    total = length(steps.fetch_orders.result)
    data  = steps.fetch_orders.result
  }
}
```
