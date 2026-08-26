endpoint "GET /users/me" {
  description = "Fetches the current authenticated user record based on JWT claims."

  pipeline {
    sql "find_user" {
      connection = connection.postgres.replica
      query      = <<-SQL
        SELECT id, name, email, created_at
        FROM users
        WHERE id = @id
      SQL
      args = {
        id = ctx.auth.claims.sub
      }
    }

    respond {
      condition = steps.find_user.rows_affected == 0
      status    = 404
      body      = { error = "User identity not found" }
    }

    respond {
      status = 200
      body   = steps.find_user.result
    }
  }
}