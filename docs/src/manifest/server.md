# server

Declares transport-level settings: listener interface and port, connection
timeouts, and the maximum request body size.

## Declaration

```hcl
server {
  host          = "0.0.0.0"
  port          = 8080
  read_timeout  = "15s"
  write_timeout = "30s"
  idle_timeout  = "60s"
  max_body_size = "25MB"
}
```

## Attributes

All attributes are optional. Omitted attributes take the default listed.

| Attribute       | Type       | Default       | Description                                         |
| :-------------- | :--------- | :------------ | :-------------------------------------------------- |
| `host`          | `string`   | `"127.0.0.1"` | interface to bind; `"0.0.0.0"` binds all interfaces |
| `port`          | `int`      | `8080`        | TCP port                                            |
| `read_timeout`  | `Duration` | `"15s"`       | maximum time to read the full request               |
| `write_timeout` | `Duration` | `"15s"`       | maximum time to write the response                  |
| `idle_timeout`  | `Duration` | `"60s"`       | maximum idle time on a keep-alive connection        |
| `max_body_size` | `ByteSize` | `"10MB"`      | requests larger than this are rejected at ingress   |

`Duration` and `ByteSize` are defined in [Scalar types](./types.md).

## Examples

A minimal local server.

```hcl
server {}

endpoint "GET /ping" {
  pipeline {
    respond {
      status = 200
      body   = { status = "pong" }
    }
  }
}
```

A production container configuration.

```hcl
server {
  host          = "0.0.0.0"
  port          = 8080
  read_timeout  = "30s"
  write_timeout = "60s"
  idle_timeout  = "120s"
  max_body_size = "50MB"
}
```

## Precedence

Runtime values override the manifest without requiring a rebuild:

```text
1. CLI flags              (--port 9000 --host 0.0.0.0)
2. Environment variables  (HCLAPI_PORT, PORT, HCLAPI_HOST, HOST)
3. Manifest (server { })
4. Built-in defaults
```

```sh
PORT=3000 HOST=0.0.0.0 hclapi serve -c ./config
hclapi serve -c . --port 9000
```
