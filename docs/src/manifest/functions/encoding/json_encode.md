# json_encode

Serializes any HCL data structure (primitive, map, object, list) into a valid JSON string.

## Signature

```hcl
json_encode(value: any) -> string
```

## Parameters

| Parameter | Type  | Required | Description                              |
| :-------- | :---- | :------- | :--------------------------------------- |
| `value`   | `any` | yes      | The value or data structure to serialize |

## Return value

Returns a JSON-formatted `string`.

## Examples

### Writing a query result to redis

```hcl
endpoint "GET /api/v1/products/{id}" {
  pipeline {
    sql "find_product" {
      connection = connection.postgres.main
      query      = "SELECT id, name, price FROM products WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    redis "cache_write" {
      connection = connection.redis.cache
      command    = "SET"
      key        = "product:${ctx.request.path.id}"
      value      = json_encode(steps.find_product.result)
      ttl        = "30m"
    }

    respond {
      status = 200
      body   = steps.find_product.result
    }
  }
}
```
