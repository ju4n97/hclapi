# now

Returns the current system timestamp in UTC formatted according to RFC 3339.

## Signature

```hcl
now() -> string
```

## Parameters

None.

## Return value

Returns an RFC 3339 formatted timestamp `string` (e.g. `"2026-08-29T22:00:00Z"`).

## Examples

```hcl
endpoint "GET /api/v1/ping" {
  pipeline {
    respond {
      status = 200
      body = {
        status      = "ok"
        received_at = now()
        epoch       = ctx.timestamp_epoch
      }
    }
  }
}
```
