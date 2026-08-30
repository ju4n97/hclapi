---
title: hmac_sha256
description: Compute an HMAC-SHA256 signature for a payload using a shared secret key.
---

# hmac_sha256

Computes an HMAC-SHA256 signature for a payload using a shared secret key. Returns a lowercase hexadecimal string.

## Signature

```hcl
hmac_sha256(key: string, message: string) -> string
```

## Parameters

| Parameter | Type     | Required | Description                     |
| :-------- | :------- | :------- | :------------------------------ |
| `key`     | `string` | yes      | The secret key used for signing |
| `message` | `string` | yes      | The message payload to sign     |

## Return value

Returns a 64-character hexadecimal signature `string`.

## Examples

```hcl
respond {
  condition = ctx.request.headers.x_hub_signature != hmac_sha256(env("WEBHOOK_SECRET"), json_encode(ctx.request.body))
  status    = 401
  body      = { error = "Invalid webhook signature" }
}
```
