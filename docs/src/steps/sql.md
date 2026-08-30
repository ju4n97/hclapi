# sql

Executes a parameterized query or mutation against a database connection.
Parameters are bound through prepared statements.

## Declaration

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

## Attributes

| Attribute        | Type         | Required | Description                                          |
| :--------------- | :----------- | :------- | :--------------------------------------------------- |
| label            | `string`     | yes      | step identifier; output is written to `steps.<name>` |
| `connection`     | `connection` | yes      | database connection pool                             |
| `query`          | `string`     | yes      | SQL statement; parameters use `@name`                |
| `args`           | `map`        | no       | binds context values to `@parameters`                |
| `catch "<code>"` | `block`      | no       | handles a specific database error code               |

## Output

| Field                        | Type  | Description                                                      |
| :--------------------------- | :---- | :--------------------------------------------------------------- |
| `steps.<name>.result`        | `any` | a list for multiple rows, an object for one row, `null` for none |
| `steps.<name>.rows_affected` | `int` | rows returned, updated, deleted, or inserted                     |

## Catching errors

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

See [404 on a missing record](../patterns.md#404-on-a-missing-record) for
the `rows_affected` idiom.
