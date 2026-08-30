# trim_prefix

Removes a prefix from the start of a string if present.

## Signature

```hcl
trim_prefix(str: string, prefix: string) -> string
```

## Examples

```hcl
sql "validate_token" {
  connection = connection.postgres.main
  query      = "SELECT user_id FROM tokens WHERE token = @token"
  args = {
    token = trim_prefix(ctx.request.headers.authorization, "Bearer ")
  }
}
```
