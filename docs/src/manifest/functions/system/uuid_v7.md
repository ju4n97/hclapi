# uuid_v7

Generates a time-ordered UUID version 7 string (RFC 9562). UUID v7 combines a Unix millisecond timestamp with random entropy, preventing B-Tree index fragmentation and page thrashing in database primary keys.

## Signature

```hcl
uuid_v7() -> string
```

## Parameters

None.

## Return value

Returns a 36-character hyphenated time-sortable UUID `string`.

## Examples

### Primary key generation for database inserts

```hcl
endpoint "POST /api/v1/orders" {
  pipeline {
    sql "create_order" {
      connection = connection.postgres.main
      query      = <<-SQL
        INSERT INTO orders (id, user_id, amount)
        VALUES (@id, @user_id, @amount)
        RETURNING id, created_at
      SQL
      args = {
        id      = uuid_v7()
        user_id = ctx.request.body.user_id
        amount  = ctx.request.body.amount
      }
    }

    respond {
      status = 201
      body   = steps.create_order.result
    }
  }
}
```
