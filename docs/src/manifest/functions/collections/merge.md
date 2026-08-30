# merge

Merges two or more maps into a single map. If duplicate keys exist, values from later arguments take precedence.

## Signature

```hcl
merge(...maps: map) -> map
```

## Examples

```hcl
respond {
  status = 200
  body = merge(steps.find_user.result, {
    retrieved_at = ctx.timestamp_epoch
    source_ip    = ctx.request.headers.x_forwarded_for
  })
}
```
