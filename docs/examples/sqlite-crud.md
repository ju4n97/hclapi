# Example 02: SQLite CRUD

Demonstrates complete CRUD operations backed by an embedded SQLite database with schema validation and conditional 404 responses.

## Manifest specification

```hcl
connection "sqlite" "main" {
  url = "file:todos.db?cache=shared&mode=rwc"

  pool {
    max_open_conns = 1
    idle_timeout   = "10m"
  }
}

schema "todo_create" {
  field "title" {
    type     = string
    required = true
  }
}

endpoint "GET /api/v1/todos" {
  pipeline {
    sql "list" {
      connection = connection.sqlite.main
      query      = <<-SQL
        SELECT id, title, completed, created_at
        FROM todos
        ORDER BY id DESC
      SQL
    }

    respond {
      status = 200
      body   = steps.list.result
    }
  }
}

endpoint "POST /api/v1/todos" {
  request {
    body = schema.todo_create
  }

  pipeline {
    sql "insert" {
      connection = connection.sqlite.main
      query      = <<-SQL
        INSERT INTO todos (title)
        VALUES (@title)
        RETURNING id, title, completed, created_at
      SQL
      args = {
        title = ctx.request.body.title
      }
    }

    respond {
      status = 201
      body   = steps.insert.result
    }
  }
}
```
