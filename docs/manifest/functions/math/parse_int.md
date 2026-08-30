---
title: parse_int
description: Parse an integer from a string representation in a base from 2 to 36.
---

# parse_int

Parses an integer from a string representation in the specified base (2 to 36).

## Signature

```hcl
parse_int(str: string, base: int) -> int
```

## Parameters

| Parameter | Type     | Required | Description                 |
| :-------- | :------- | :------- | :-------------------------- |
| `str`     | `string` | yes      | The numeric string to parse |
| `base`    | `int`    | yes      | Radix base between 2 and 36 |

## Return value

Returns the parsed integer value.

## Examples

### Parsing hexadecimal identifiers

```hcl
sql "find_by_bitmask" {
  connection = connection.postgres.main
  query      = "SELECT id FROM records WHERE flags = @mask"
  args = {
    mask = parse_int(ctx.request.query.hex_flag, 16)
  }
}
```
