---
title: server
description: Transport-level listener, timeout, request-size, and error-documentation settings.
---

# server

Declares transport-level settings: listener interface and port, connection timeouts, maximum request body size, and error documentation URIs.

## Declaration

```hcl
server {
  host           = "0.0.0.0"
  port           = 8080
  read_timeout   = "15s"
  write_timeout  = "30s"
  idle_timeout   = "60s"
  max_body_size  = "25MB"
  error_base_url = "https://docs.mycompany.com/errors/"
}
```

## Attributes

All attributes are optional. Omitted attributes take the default listed.

| Attribute        | Type       | Default       | Description                                          |
| :--------------- | :--------- | :------------ | :--------------------------------------------------- |
| `host`           | `string`   | `"127.0.0.1"` | Interface to bind; `"0.0.0.0"` binds all interfaces  |
| `port`           | `int`      | `8080`        | TCP port                                             |
| `read_timeout`   | `Duration` | `"15s"`       | Maximum time to read the full request                |
| `write_timeout`  | `Duration` | `"15s"`       | Maximum time to write the response                   |
| `idle_timeout`   | `Duration` | `"60s"`       | Maximum idle time on a keep-alive connection         |
| `max_body_size`  | `ByteSize` | `"10MB"`      | Requests larger than this are rejected with HTTP 413 |
| `error_base_url` | `string`   | `""`          | Base URI prefix for RFC 9457 error problem types     |

## Error documentation URI resolution

When returning RFC 9457 Problem Details error responses:

- If `error_base_url` is omitted: The engine uses standard URN identifiers (e.g. `"urn:hclapi:error:bad-request"`).
- If `error_base_url` is set: The engine prefixes the error slug (e.g. `"https://docs.mycompany.com/errors/bad-request"`).

## Examples

### Production container configuration

```hcl
server {
  host           = "0.0.0.0"
  port           = 8080
  read_timeout   = "30s"
  write_timeout  = "60s"
  idle_timeout   = "120s"
  max_body_size  = "50MB"
  error_base_url = "https://developer.example.com/api/errors/"
}
```
