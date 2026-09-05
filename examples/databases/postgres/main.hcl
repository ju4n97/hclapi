server {
  host = "127.0.0.1"
  port = 8080

  openapi {
    title       = "PostgreSQL Members API"
    version     = "1.0.0"
    description = "Complete CRUD API backed by PostgreSQL with stored procedure execution."
  }
}

connection "postgres" "main" {
  source = "postgres://hclapi:hclapi_password@127.0.0.1:5432/hclapi_db?sslmode=disable"

  pool {
    max_open     = 25
    max_idle     = 5
    max_lifetime = "30m"
    idle_timeout = "5m"
  }
}

schema "member_create" {
  field "name" {
    type       = string
    required   = true
    min_length = 2
  }

  field "email" {
    type     = string
    required = true
    format   = "email"
  }

  field "tier" {
    type    = string
    default = "standard"
    enum    = ["standard", "premium", "enterprise"]
  }
}

schema "member_update" {
  field "name" {
    type       = string
    min_length = 2
  }

  field "tier" {
    type = string
    enum = ["standard", "premium", "enterprise"]
  }
}

schema "reward_points" {
  field "bonus" {
    type     = int
    required = true
    min      = 1
    max      = 10000
  }
}

endpoint "GET /docs" {
  openapi {
    ui = "scalar"
  }
}

endpoint "GET /openapi.json" {
  openapi {
    format = "json"
  }
}

endpoint "GET /api/v1/members" {
  description = "Lists all registered members."

  pipeline {
    sql "list" {
      connection = connection.postgres.main
      query      = "SELECT id, name, email, tier, points, created_at FROM members ORDER BY id DESC"
    }

    respond {
      status = 200
      body   = steps.list.rows
    }
  }
}

endpoint "POST /api/v1/members" {
  description = "Creates a new member and catches unique constraint violations."

  request {
    body = schema.member_create
  }

  pipeline {
    sql "insert" {
      connection = connection.postgres.main
      query      = <<-SQL
        INSERT INTO members (name, email, tier)
        VALUES (@name, @email, @tier)
        RETURNING id, name, email, tier, points, created_at
      SQL
      args = {
        name  = ctx.request.body.name
        email = ctx.request.body.email
        tier  = ctx.request.body.tier
      }

      catch "23505" {
        status = 409
        body   = problem(409, "Email address already registered", "email-collision")
      }
    }

    respond {
      status = 201
      body   = steps.insert.row
    }
  }
}

endpoint "GET /api/v1/members/{id}" {
  description = "Fetches a single member by ID."

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
      connection = connection.postgres.main
      query      = "SELECT id, name, email, tier, points, created_at FROM members WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    respond {
      condition = steps.fetch.rows_affected == 0
      status    = 404
      body      = problem(404, "Member ${ctx.request.path.id} not found")
    }

    respond {
      status = 200
      body   = steps.fetch.row
    }
  }
}

endpoint "PUT /api/v1/members/{id}" {
  description = "Updates an existing member."

  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }
    body = schema.member_update
  }

  pipeline {
    sql "update" {
      connection = connection.postgres.main
      query      = <<-SQL
        UPDATE members
        SET
          name = COALESCE(@name, name),
          tier = COALESCE(@tier, tier)
        WHERE id = @id
        RETURNING id, name, email, tier, points, created_at
      SQL
      args = {
        id   = ctx.request.path.id
        name = ctx.request.body.name
        tier = ctx.request.body.tier
      }
    }

    respond {
      condition = steps.update.rows_affected == 0
      status    = 404
      body      = problem(404, "Member ${ctx.request.path.id} not found")
    }

    respond {
      status = 200
      body   = steps.update.row
    }
  }
}

endpoint "DELETE /api/v1/members/{id}" {
  description = "Deletes a member record."

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
      connection = connection.postgres.main
      query      = "DELETE FROM members WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    respond {
      condition = steps.delete.rows_affected == 0
      status    = 404
      body      = problem(404, "Member ${ctx.request.path.id} not found")
    }

    respond {
      status = 204
    }
  }
}

endpoint "POST /api/v1/members/{id}/reward" {
  description = "Executes a PostgreSQL stored procedure to award points."

  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }
    body = schema.reward_points
  }

  pipeline {
    sql "award" {
      connection = connection.postgres.main
      query      = "CALL award_member_points(@id, @bonus)"
      args = {
        id    = ctx.request.path.id
        bonus = ctx.request.body.bonus
      }
    }

    respond {
      status = 200
      body = {
        message   = "Points awarded successfully",
        member_id = ctx.request.path.id,
        awarded   = ctx.request.body.bonus
      }
    }
  }
}