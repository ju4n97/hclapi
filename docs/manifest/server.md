---
title: Server block
description: Configuration reference for the global HTTP server block, network bindings, timeouts, and payload limits
---

# Server block

The `server` block defines global transport-level HTTP listener parameters, including network interfaces, TCP port bindings, connection timeouts, and request payload size limits.

## Block declaration

A `server` block is declared at the root level of a manifest:

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

## Attribute reference

All attributes in the `server` block are optional. When omitted, the engine applies baseline production defaults.

| Attribute           | Type       | Default       | Description                                                                                     |
| :------------------ | :--------- | :------------ | :---------------------------------------------------------------------------------------------- |
| **`host`**          | `string`   | `"127.0.0.1"` | Network IP interface address to bind the listener. Use `"0.0.0.0"` to listen on all interfaces. |
| **`port`**          | `int`      | `8080`        | TCP port number for the HTTP server.                                                            |
| **`read_timeout`**  | `Duration` | `"15s"`       | Maximum duration allowed for reading the entire incoming request, including headers and body.   |
| **`write_timeout`** | `Duration` | `"15s"`       | Maximum duration allowed before timing out writes of the response.                              |
| **`idle_timeout`**  | `Duration` | `"60s"`       | Maximum duration to wait for the next request when HTTP keep-alive connections are enabled.     |
| **`max_body_size`** | `ByteSize` | `"10MB"`      | Maximum allowable request body size. Payloads exceeding this threshold are rejected at ingress. |

## Scalar types in server configuration

The `server` block uses strongly-typed scalar definitions that deserialize human-readable strings into Go types at parse time:

- **`Duration`**: Accepts time duration strings (e.g. `"500ms"`, `"15s"`, `"1m"`, `"1h30m"`).
- **`ByteSize`**: Accepts binary and decimal byte units (e.g. `"512B"`, `"10KB"`, `"10KiB"`, `"25MB"`, `"1GB"`).

Detailed specifications for scalar types are documented in [Scalar types](/manifest/types).

## Configuration examples

### 1. Minimal local development server

Omits explicit attributes to inherit local loopback defaults (`127.0.0.1:8080`):

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

### 2. Production container configuration

Binds to all interfaces with extended write timeouts for database aggregation and a 50 megabyte payload threshold:

```hcl
server {
  host          = "0.0.0.0"
  port          = 8080
  read_timeout  = "30s"
  write_timeout = "60s"
  idle_timeout  = "120s"
  max_body_size = "50MB"
}

endpoint "POST /api/v1/uploads" {
  pipeline {
    # Processing pipeline
    respond {
      status = 200
      body   = { received = true }
    }
  }
}
```

## Runtime precedence and overrides

Settings defined in the `server` block serve as the configuration baseline, but can be overridden at runtime without modifying the manifest files.

The engine resolves server settings using the following precedence hierarchy:

```text
1. CLI Flags              (--port 9000 --host 0.0.0.0)
2. Environment Variables  (HCLAPI_PORT, PORT, HCLAPI_HOST, HOST)
3. Manifest Configuration (server { ... } block)
4. Built-in Defaults      (127.0.0.1:8080, 15s timeouts, 10MB limit)
```

### Overriding via environment variables in containerized environments

When deployed in container platforms (such as Kubernetes, AWS ECS, or Google Cloud Run), the target port can be assigned dynamically:

```bash
# Binds to port 3000 and all network interfaces
PORT=3000 HOST=0.0.0.0 hclapi serve -c ./config
```

### Overriding via command-line flags

```bash
# Explicitly overrides the port defined in Hclapifile
hclapi serve -c . --port 9000
```
