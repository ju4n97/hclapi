---
title: Expressions
description: Request-time HCL expressions for context access, conditions, interpolation, argument mapping, and response construction.
---

# Expressions

HCL attributes support expressions evaluated at request time. Expressions read from [`ctx`](./context.md) to map parameters, branch on conditions, interpolate strings, and construct response bodies.

## Operators

| Operator             | Meaning            | Example                                                                     |
| :------------------- | :----------------- | :-------------------------------------------------------------------------- |
| `==`, `!=`           | equality           | `steps.lookup.rows_affected == 0`                                           |
| `>`, `>=`, `<`, `<=` | numeric comparison | `steps.inventory.result.count < 5`                                          |
| `&&`                 | logical AND        | `steps.auth.result.valid == true && ctx.request.body.admin == true`         |
| `\|\|`               | logical OR         | `ctx.request.query.format == "csv" \|\| ctx.request.query.format == "xlsx"` |
| `!`                  | logical NOT        | `!steps.user.result.is_active`                                              |

A field that may be absent is compared against `null`:

```hcl
respond {
  condition = steps.cache_lookup.result != null
  status    = 200
  body      = steps.cache_lookup.result
}
```

## String interpolation

`${...}` substitutes a value from `ctx.request` or `steps`. Multiple substitutions may appear in one string.

```hcl
redis "session_write" {
  connection = connection.redis.sessions
  command    = "SET"
  key        = "session:${ctx.request.headers.x_session_id}:user"
  value      = steps.find_user.result.id
  ttl        = "15m"
}
```

## Argument mapping

`args` maps context values to parameterized query inputs. Parameters bound through `@param` in a `sql` block are sanitized automatically.

```hcl
sql "update_account" {
  connection = connection.postgres.main
  query      = <<-SQL
    UPDATE accounts
    SET name = @name, updated_at = NOW()
    WHERE id = @id
    RETURNING id, name, updated_at
  SQL
  args = {
    id   = ctx.request.path.id
    name = steps.sanitize_input.result.clean_name
  }
}
```

## Response bodies

Object and list literals may be constructed inline.

```hcl
respond {
  status = 201
  body = {
    account = steps.create_account.result
    metadata = {
      requested_by = ctx.request.headers.authorization
      created_at   = ctx.timestamp_epoch
      tags         = ["api", "v1", ctx.request.query.environment]
    }
  }
}
```
