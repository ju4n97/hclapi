endpoint "POST /api/v1/users" {
  description = "Registers a new user."

  request {
    body = schema.user_create
  }

  pipeline {
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          return {
            "email": ctx.request.body.get("email", "").strip().lower(),
            "name": ctx.request.body.get("full_name", "").strip()
          }
      STARLARK
    }

    sql "insert" {
      connection = connection.sqlite.main
      query      = <<-SQL
        INSERT INTO users (email, name)
        VALUES (@email, @name)
        RETURNING id, email, name, created_at
      SQL

      args = {
        email = steps.normalize.result.email
        name  = steps.normalize.result.name
      }

      catch "19" {
        status = 409
        body   = problem(409, "User email already exists", "email-collision")
      }
    }

    respond {
      status = 201
      body   = steps.insert.row
    }
  }
}