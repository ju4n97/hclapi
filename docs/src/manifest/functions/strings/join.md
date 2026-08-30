# join

Concatenates elements in a list of strings using a `separator`.

## Signature

```hcl
join(separator: string, list: list(string)) -> string
```

## Examples

```hcl
redis "audit_log" {
  connection = connection.redis.cache
  command    = "SET"
  key        = "log:${ctx.timestamp_epoch}"
  value      = join(":", [ctx.request.method, ctx.request.path.id, "COMPLETED"])
}
```
