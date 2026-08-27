---
title: Scalar types
description: Syntax, units, and deserialization rules for duration and byte size scalar types in Hclapi manifests.
---

# Scalar types

Hclapi provides specialized scalar types that deserialize human-readable strings into typed representations during manifest compilation. These types eliminate manual unit parsing while providing strict parse-time validation.

## Deserialization mechanism

Scalar types implement standard Go `encoding.TextUnmarshaler` interfaces. When the parser encounters an attribute assigned a scalar type, the string literal is validated and coerced before the server opens network listeners.

If an invalid unit, negative value, or malformed string is supplied, compilation halts immediately with a file, line, and column diagnostic.

## Duration specification

The `Duration` type represents elapsed time. It is backed by Go's standard `time.Duration` nanosecond representation.

### Supported time units

| Unit suffix | Meaning      | Example          |
| :---------- | :----------- | :--------------- |
| `ns`        | Nanoseconds  | `"500ns"`        |
| `us`, `µs`  | Microseconds | `"100us"`        |
| `ms`        | Milliseconds | `"250ms"`        |
| `s`         | Seconds      | `"15s"`, `"30s"` |
| `m`         | Minutes      | `"5m"`, `"30m"`  |
| `h`         | Hours        | `"1h"`, `"24h"`  |

### Compound durations

Multiple unit qualifiers can be combined into a single duration literal:

```hcl
read_timeout = "1h30m"
idle_timeout = "2m45s"
```

## ByteSize specification

The `ByteSize` type represents memory or storage quantity in integer bytes. It is backed by a 64-bit signed integer (`int64`).

### Supported byte units

Both binary (1024-based) and decimal (1000-based) unit qualifiers are supported. Matching is case-insensitive.

| Unit suffix | Standard         | Multiplier | Exact bytes         |
| :---------- | :--------------- | :--------- | :------------------ |
| `B`         | Single Byte      | $1$        | `1`                 |
| `KB`, `K`   | Decimal Kilobyte | $1000^1$   | `1,000`             |
| `KiB`       | Binary Kibibyte  | $1024^1$   | `1,024`             |
| `MB`, `M`   | Decimal Megabyte | $1000^2$   | `1,000,000`         |
| `MiB`       | Binary Mebibyte  | $1024^2$   | `1,048,576`         |
| `GB`, `G`   | Decimal Gigabyte | $1000^3$   | `1,000,000,000`     |
| `GiB`       | Binary Gibibyte  | $1024^3$   | `1,073,741,824`     |
| `TB`, `T`   | Decimal Terabyte | $1000^4$   | `1,000,000,000,000` |
| `TiB`       | Binary Tebibyte  | $1024^4$   | `1,099,511,627,776` |

### Fractional and raw integer notation

Byte sizes support decimal floating-point prefixes as well as raw byte counts:

```hcl
max_body_size = "1.5GB"     # 1,500,000,000 bytes
max_body_size = "2.5MiB"    # 2,621,440 bytes
max_body_size = "1048576"   # 1,048,576 bytes (raw integer fallback)
```

## Example usage in manifest definitions

```hcl
server {
  host          = "0.0.0.0"
  port          = 8080
  read_timeout  = "15s"     # Duration
  write_timeout = "30s"     # Duration
  idle_timeout  = "2m"      # Duration
  max_body_size = "25MB"    # ByteSize
}

connection "postgres" "primary" {
  url = env("DATABASE_PRIMARY_URL")

  pool {
    max_open_conns    = 50
    max_idle_conns    = 10
    conn_max_lifetime = "1h"    # Duration
    idle_timeout      = "10m"   # Duration
  }
}

endpoint "POST /api/v1/cache" {
  pipeline {
    redis "store_session" {
      connection = connection.redis.cache
      command    = "SET"
      key        = "session:${ctx.request.headers.session_id}"
      value      = ctx.request.body.token
      ttl        = "30m"    # Duration
    }

    respond {
      status = 200
      body   = { stored = true }
    }
  }
}
```

## Parse-time validation diagnostics

Invalid string units or unparseable formats cause the compilation phase to fail before the server begins listening for traffic:

### Invalid duration syntax

```hcl
# Invalid: '100years' is not a recognized time unit
read_timeout = "100years"
```

**Diagnostic output:**

```text
failed to decode HCL file server.hcl: invalid duration "100years": time: unknown unit "years" in duration "100years"
```

### Invalid byte size syntax

```hcl
# Invalid: 'XB' is not a recognized byte unit
max_body_size = "10XB"
```

**Diagnostic output:**

```text
failed to decode HCL file server.hcl: invalid byte size "10XB": invalid unit "XB" in byte size "10XB"
```
