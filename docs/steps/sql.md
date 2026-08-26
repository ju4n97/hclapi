# SQL step

The `sql` step executes parameterized queries against a defined database connection pool.

## Syntax

```hcl
sql "insert_user" {
  connection = connection.postgres.main
  query      = <<-SQL
    INSERT INTO users (name, email)
    VALUES (@name, @email)
    RETURNING id, name, email, created_at
  SQL

  args = {
    name  = steps.normalize.result.name
    email = steps.normalize.result.email
  }

  catch "23505" {
    abort_with_status = 409
    body = { error = "Email address already registered" }
  }
}
```

## Parameter substitution rules

1. Named parameters use the `@param` prefix within SQL strings.
2. The engine maps named parameters to driver-specific placeholders (`$1` for PostgreSQL, `?` for SQLite/MySQL, `@p1` for MSSQL).
3. Dynamic string interpolation (`${...}`) inside the `query` attribute is rejected at compile time to guarantee SQL injection immunity.
