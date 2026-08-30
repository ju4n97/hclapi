# json_decode

Parses a JSON string into corresponding HCL primitives, lists, or maps.

## Signature

```hcl
json_decode(str: string) -> any
```

## Parameters

| Parameter | Type     | Required | Description              |
| :-------- | :------- | :------- | :----------------------- |
| `str`     | `string` | yes      | The JSON string to parse |

## Return value

Returns parsed JSON as an HCL object, list, or primitive.

## Examples

### Reading and decoding cached data

```hcl
endpoint "GET /api/v1/products/{id}" {
  pipeline {
    redis "cache_lookup" {
      connection = connection.redis.cache
      command    = "GET"
      key        = "product:${ctx.request.path.id}"
    }

    respond {
      condition = steps.cache_lookup.result != null
      status    = 200
      headers   = { "X-Cache" = "HIT" }
      body      = json_decode(steps.cache_lookup.result)
    }

    respond {
      status = 404
      body   = { error = "Product not found in cache" }
    }
  }
}
```
