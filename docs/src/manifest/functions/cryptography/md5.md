# md5

Computes the MD5 checksum of a string, returning a 32-character hexadecimal digest.

## Signature

```hcl
md5(str: string) -> string
```

## Parameters

| Parameter | Type     | Required | Description              |
| :-------- | :------- | :------- | :----------------------- |
| `str`     | `string` | yes      | The input string to hash |

## Return value

Returns a 32-character hexadecimal `string`.

## Examples

```hcl
respond {
  status = 200
  headers = {
    "ETag" = format("\"%s\"", md5(json_encode(steps.find_user.result)))
  }
  body = steps.find_user.result
}
```
