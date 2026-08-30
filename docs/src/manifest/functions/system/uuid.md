# uuid / uuid_v4

Generates a cryptographically secure random UUID version 4 string. `uuid()` is an alias to `uuid_v4()`.

## Signature

```hcl
uuid() -> string
uuid_v4() -> string
```

## Parameters

None.

## Return value

Returns a 36-character hyphenated UUID `string` (e.g. `"f47ac10b-58cc-4372-a567-0e02b2c3d479"`).

## Examples

### Setting a request trace header

```hcl
endpoint "POST /api/v1/sessions" {
  pipeline {
    respond {
      status = 201
      headers = {
        "X-Request-ID" = uuid()
      }
      body = {
        session_id = uuid_v4()
        status     = "created"
      }
    }
  }
}
```

## See also

- [`uuid_v7`](./uuid_v7.md) for time-ordered UUID version 7 string
