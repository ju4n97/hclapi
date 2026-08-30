# base64_encode

Encodes a string using standard RFC 4648 Base64 encoding.

## Signature

```hcl
base64_encode(str: string) -> string
```

## Parameters

| Parameter | Type     | Required | Description                     |
| :-------- | :------- | :------- | :------------------------------ |
| `str`     | `string` | yes      | The plain text string to encode |

## Return value

Returns a Base64 encoded `string`.

## Examples

```hcl
endpoint "POST /api/v1/credentials" {
  pipeline {
    respond {
      status = 200
      body = {
        basic_token = base64_encode("${ctx.request.body.user}:${ctx.request.body.pass}")
      }
    }
  }
}
```
