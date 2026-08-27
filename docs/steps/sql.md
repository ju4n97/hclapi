---
title: SQL
description: Parameterized SQL execution across PostgreSQL, SQLite, and relational database connection pools.
---

# SQL

The `sql` step executes parameterized queries and transactions against relational database connection pools. Statements support named argument binding, automatic SQL injection prevention, execution metrics tracking, and database error code mapping.

## Block declaration

```hcl
sql "<name>" {
  connection = connection.<driver>.<name>
  query      = <<-SQL
    SELECT id, name, email FROM users WHERE id = @id
  SQL
  args = {
    id = ctx.request.path.id
  }
}
```

## Attribute reference

| Attribute | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| **`name`** (Label) | `string` | Yes | Unique step identifier used under `steps.<name>.result`. |
| **`connection`** | `connection` | Yes | Reference to a declared database connection pool. |
| **`query`** | `string` | Yes | Parameterized SQL statement. Parameters use the `@<name>` syntax. |
| **`args`** | `map` | No | Map binding execution context variables to query `@parameters`. |
| **`catch "<code/error>"`** | `block` | No | Error handling block for specific database error codes or constraints. |

## Parameter binding syntax (`@param`)

Query variables use the `@param_name` notation. The engine sanitizes and binds these parameters through prepared statements, preventing SQL injection attacks:

```hcl
sql "find_product" {
  connection = connection.postgres.main
  query      = <<-SQL
    SELECT id, sku, name, price_cents, inventory
    FROM products
    WHERE sku = @sku AND active = @is_active
  SQL
  args = {
    sku       = ctx.request.path.sku
    is_active = true
  }
}
```

## Output metrics and results

Executing an `sql` step populates two context fields:

| Field | Type | Description |
| :--- | :--- | :--- |
| `steps.<name>.result` | `any` | Result set. Returns a list of record objects for multiple rows, a single object for unique rows, or `null` if no records match. |
| `steps.<name>.rows_affected` | `int` | Number of rows updated, deleted, inserted, or returned by the query. |

### Using `rows_affected` for 404 detection

```hcl
endpoint "GET /api/v1/users/{id}" {
  pipeline {
    sql "find_user" {
      connection = connection.postgres.main
      query      = "SELECT id, name, email FROM users WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    # If no rows match, return 404 immediately
    respond {
      condition = steps.find_user.rows_affected == 0
      status    = 404
      body      = { error = "User not found" }
    }

    respond {
      status = 200
      body   = steps.find_user.result
    }
  }
}
```

## Error code catching (`catch`)

Database error codes (such as PostgreSQL unique violation `23505`) can be caught and mapped directly to HTTP error responses:

```hcl
sql "insert_user" {
  connection = connection.postgres.main
  query      = <<-SQL
    INSERT INTO users (email, full_name)
    VALUES (@email, @full_name)
    RETURNING id, email, full_name, created_at
  SQL
  args = {
    email     = ctx.request.body.email
    full_name = ctx.request.body.full_name
  }

  # PostgreSQL unique constraint violation code
  catch "23505" {
    abort_with_status = 409
    body = {
      type   = "https://hclapi.dev/errors/conflict"
      title  = "Conflict"
      status = 409
      detail = "A user with this email address already exists."
    }
  }
}
```
