# contains

Determines whether a list contains a specific value.

## Signature

```hcl
contains(list: list, value: any) -> bool
```

## Examples

```hcl
respond {
  condition = !contains(["admin", "editor"], ctx.request.headers.x_role)
  status    = 403
  body      = { error = "Forbidden: Insufficient privileges" }
}
```
