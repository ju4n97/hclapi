# Global configuration

Top-level singleton blocks define operational parameters for the standalone runtime.

## The server block

Configures the standalone HTTP socket listener:

```hcl
server {
  host = "0.0.0.0"
  port = 8080

  read_timeout  = "15s"
  write_timeout = "15s"
  idle_timeout  = "60s"
}
```

### Attributes

| Attribute | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `host` | `string` | `"0.0.0.0"` | Network interface IP address to bind. |
| `port` | `int` | `8080` | TCP listening port. |
| `read_timeout` | `duration` | `"10s"` | Maximum duration for reading the entire request. |
| `write_timeout` | `duration` | `"10s"` | Maximum duration before timing out response writes. |
| `idle_timeout` | `duration` | `"60s"` | Maximum duration to wait for the next request on keep-alive. |

::: info
When Hclapi is embedded as a Go library via `hclapi.NewEngine`, the `server` block is ignored because the host application manages the HTTP listener.
:::

## The api block

Defines global path prefixes and default authentication schemes:

```hcl
api {
  prefix = "/api/v1"
  auth   = [auth.jwt_main]
}
```

## Environment variable resolution

Manifests resolve host environment variables through the built-in `env()` function:

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
}
```
