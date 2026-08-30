---
title: url_encode
description: Percent-encode a string for inclusion in a URL query parameter.
---

# url_encode

Escapes characters in a string to make it safe for inclusion inside a URL query parameter.

## Signature

```hcl
url_encode(str: string) -> string
```

## Parameters

| Parameter | Type     | Required | Description          |
| :-------- | :------- | :------- | :------------------- |
| `str`     | `string` | yes      | The string to escape |

## Return value

Returns a URL percent-encoded `string`.

## Examples

```hcl
endpoint "GET /api/v1/auth/redirect" {
  pipeline {
    respond {
      status = 302
      headers = {
        "Location" = "https://idp.example.com/oauth?redirect_uri=${url_encode(ctx.request.query.callback)}"
      }
    }
  }
}
```
