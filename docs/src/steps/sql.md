# sql

Executes a parameterized SQL query or mutation against a database connection pool. Parameters are bound through prepared statements using `@name` placeholders.

## Declaration

```hcl
sql "find_user" {
  connection = connection.postgres.main
  query      = <<-SQL
    SELECT id, name, email FROM users WHERE id = @id
  SQL
  args = {
    id = ctx.request.path.id
  }
}
```

## Attributes

| Attribute        | Type         | Required | Description                                            |
| :--------------- | :----------- | :------- | :----------------------------------------------------- |
| label            | `string`     | yes      | Step identifier; outputs are written to `steps.<name>` |
| `connection`     | `connection` | yes      | Database connection pool reference                     |
| `query`          | `string`     | yes      | SQL query; parameters use `@name` placeholders         |
| `args`           | `map`        | no       | Binds context expressions to `@name` parameters        |
| `catch "<code>"` | `block`      | no       | Handles specific database error codes                  |

## Exported outputs

Unlike untyped query drivers, `sql` steps export deterministic attributes to prevent JSON array/object polymorphism bugs:

| Field                        | Type            | Description                                                      |
| :--------------------------- | :-------------- | :--------------------------------------------------------------- |
| `steps.<name>.rows`          | `list(map)`     | All rows returned by the query. Returns `[]` if no rows match.   |
| `steps.<name>.row`           | `map` or `null` | The first row returned by the query, or `null` if no rows match. |
| `steps.<name>.rows_affected` | `int`           | Total count of rows matched, inserted, updated, or deleted.      |

## Catching database constraint errors

The `catch` block intercepts database error codes (such as PostgreSQL unique violation `23505` or MySQL duplicate entry `1062`), aborts any active transaction, and returns a formatted HTTP response:

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

  catch "23505" {
    status = 409
    body = {
      error = "A user with this email address already exists."
    }
  }
}
```

## Examples

### Fetching a single record

```hcl
endpoint "GET /api/v1/users/{id}" {
  pipeline {
    sql "find_user" {
      connection = connection.postgres.main
      query      = "SELECT id, name, email FROM users WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    respond {
      condition = steps.find_user.rows_affected == 0
      status    = 404
      body      = { error = "User not found" }
    }

    respond {
      status = 200
      body   = steps.find_user.row
    }
  }
}
```

### Fetching a list of records

```hcl
endpoint "GET /api/v1/users" {
  pipeline {
    sql "list_users" {
      connection = connection.postgres.main
      query      = "SELECT id, name, email FROM users ORDER BY id LIMIT 50"
    }

    respond {
      status = 200
      body   = steps.list_users.rows
    }
  }
}
```
