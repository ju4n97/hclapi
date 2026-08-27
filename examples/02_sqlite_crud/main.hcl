server {
  host = "127.0.0.1"
  port = 8080
}

connection "sqlite" "main" {
  url = "file:/data/todos.db?mode=rwc"

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

schema "todo_update" {
  field "title" {
    type = string
  }

  field "completed" {
    type = bool
  }
}

endpoint "GET /api/v1/todos" {
  description = "Lists all stored todos."

  pipeline {
    sql "list" {
      connection = connection.sqlite.main

      query = <<-SQL
        SELECT
          id,
          title,
          completed,
          created_at
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
  description = "Creates a new todo item."

  request {
    body = schema.todo_create
  }

  pipeline {
    starlark "trim_input" {
      source = <<-STARLARK
        def execute(ctx):
          return {
            "title": ctx.request.body.title.strip()
          }
      STARLARK
    }

    sql "insert" {
      connection = connection.sqlite.main

      query = <<-SQL
        INSERT INTO todos (title)
        VALUES (@title)
        RETURNING
          id,
          title,
          completed,
          created_at
      SQL

      args = {
        title = steps.trim_input.result.title
      }
    }

    respond {
      status = 201
      body   = steps.insert.result
    }
  }
}

endpoint "GET /api/v1/todos/{id}" {
  description = "Fetches a single todo by ID."

  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }
  }

  pipeline {
    sql "fetch" {
      connection = connection.sqlite.main

      query = <<-SQL
        SELECT
          id,
          title,
          completed,
          created_at
        FROM todos
        WHERE id = @id
      SQL

      args = {
        id = ctx.request.path.id
      }
    }

    respond {
      condition = steps.fetch.rows_affected == 0
      status    = 404
      body = {
        error = "Todo not found"
      }
    }

    respond {
      status = 200
      body   = steps.fetch.result
    }
  }
}

endpoint "PUT /api/v1/todos/{id}" {
  description = "Updates an existing todo."

  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }

    body = schema.todo_update
  }

  pipeline {
    sql "update" {
      connection = connection.sqlite.main

      query = <<-SQL
        UPDATE todos
        SET
          title = COALESCE(@title, title),
          completed = COALESCE(@completed, completed)
        WHERE id = @id
        RETURNING
          id,
          title,
          completed,
          created_at
      SQL

      args = {
        id        = ctx.request.path.id
        title     = ctx.request.body.title
        completed = ctx.request.body.completed
      }
    }

    respond {
      condition = steps.update.rows_affected == 0
      status    = 404
      body = {
        error = "Todo not found"
      }
    }

    respond {
      status = 200
      body   = steps.update.result
    }
  }
}

endpoint "DELETE /api/v1/todos/{id}" {
  description = "Deletes a todo item."

  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }
  }

  pipeline {
    sql "delete" {
      connection = connection.sqlite.main

      query = <<-SQL
        DELETE FROM todos
        WHERE id = @id
      SQL

      args = {
        id = ctx.request.path.id
      }
    }

    respond {
      condition = steps.delete.rows_affected == 0
      status    = 404
      body = {
        error = "Todo not found"
      }
    }

    respond {
      status = 204
    }
  }
}