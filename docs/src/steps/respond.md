# respond

Terminates the pipeline. Sets the status, headers, and body. No step after `respond` runs once it fires.

## Declaration

```hcl
respond {
  condition = steps.find_user.rows_affected == 0
  status    = 404
  headers = {
    "Cache-Control" = "no-store"
    "X-Trace-ID"    = uuid()
  }
  body = {
    error = "User not found"
  }
}
```

## Attributes

| Attribute   | Type                  | Default | Description                                           |
| :---------- | :-------------------- | :------ | :---------------------------------------------------- |
| `condition` | `Expression`          | `true`  | Step is skipped if `false`                            |
| `status`    | `int` or `Expression` | `200`   | HTTP status code                                      |
| `headers`   | `map` or `Expression` | `{}`    | Dynamic or static response headers                    |
| `body`      | `any` or `Expression` | `null`  | Payload to serialize. If omitted, no body is written. |

## Content-Type and payload serialization

- **Default JSON:** If `Content-Type` is not specified in `headers`, the engine sets `Content-Type: application/json` and JSON-encodes `body`.
- **Custom content types:** If `Content-Type` is explicitly set (e.g. `text/plain`, `text/html`, `application/xml`) and `body` is a `string` or `[]byte`, the raw payload is written directly without JSON encoding.
- **Security:** All header keys and values are sanitized to strip `\r` and `\n` characters, preventing CRLF response-splitting attacks.
