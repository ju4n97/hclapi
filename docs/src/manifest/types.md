# Scalar types

hclapi defines two scalar types that deserialize human-readable strings into
typed values at compile time. Invalid input halts compilation with a file,
line, and column diagnostic.

## Duration

Backed by Go's `time.Duration`.

| Suffix     | Unit         |
| :--------- | :----------- |
| `ns`       | nanoseconds  |
| `us`, `µs` | microseconds |
| `ms`       | milliseconds |
| `s`        | seconds      |
| `m`        | minutes      |
| `h`        | hours        |

Units may be combined.

```hcl
read_timeout = "1h30m"
idle_timeout = "2m45s"
```

## ByteSize

Backed by a 64-bit signed integer. Binary (1024-based) and decimal
(1000-based) units are both accepted; matching is case-insensitive.

| Suffix    | Standard | Bytes               |
| :-------- | :------- | :------------------ |
| `B`       | byte     | `1`                 |
| `KB`, `K` | kilobyte | `1,000`             |
| `KiB`     | kibibyte | `1,024`             |
| `MB`, `M` | megabyte | `1,000,000`         |
| `MiB`     | mebibyte | `1,048,576`         |
| `GB`, `G` | gigabyte | `1,000,000,000`     |
| `GiB`     | gibibyte | `1,073,741,824`     |
| `TB`, `T` | terabyte | `1,000,000,000,000` |
| `TiB`     | tebibyte | `1,099,511,627,776` |

Fractional prefixes and raw byte counts are both accepted.

```hcl
max_body_size = "1.5GB"     # 1,500,000,000 bytes
max_body_size = "2.5MiB"    # 2,621,440 bytes
max_body_size = "1048576"   # 1,048,576 bytes, raw integer fallback
```

## Usage

```hcl
server {
  read_timeout  = "15s"   # Duration
  max_body_size = "25MB"  # ByteSize
}

connection "postgres" "primary" {
  pool {
    conn_max_lifetime = "1h"  # Duration
  }
}
```

## Diagnostics

```hcl
read_timeout = "100years"
```

```text
failed to decode HCL file server.hcl: invalid duration "100years": time: unknown unit "years" in duration "100years"
```

```hcl
max_body_size = "10XB"
```

```text
failed to decode HCL file server.hcl: invalid byte size "10XB": invalid unit "XB" in byte size "10XB"
```
