# redis

Executes key-value operations against a Redis or Valkey connection.
Supports cache reads, writes with TTL, deletions, and counters.

## Declaration

```hcl
redis "<name>" {
  connection = connection.redis.<name>
  command    = "GET"
  key        = "cache:product:${ctx.request.path.sku}"
}
```

## Attributes

| Attribute | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| label | `string` | yes | step identifier |
| `connection` | `connection` | yes | Redis connection pool |
| `command` | `string` | yes | `GET`, `SET`, `DEL`, `INCR`, or `EXISTS` |
| `key` | `string` | yes | cache key, supports `${...}` interpolation |
| `value` | `any` | for `SET` | value to store, typically `json_encode(...)` |
| `ttl` | `Duration` | no | expiration for `SET` |

## Commands

| Command | Requires | Returns |
| :--- | :--- | :--- |
| `GET` | `key` | value, or `null` on a miss |
| `SET` | `key`, `value` | `"OK"` |
| `DEL` | `key` | number of keys removed |
| `INCR` | `key` | incremented integer |
| `EXISTS` | `key` | `bool` |

See [Cache aside](../patterns.md#cache-aside) for the full read-fallback-write
flow.
