server {
  host = "127.0.0.1"
  port = 8080

  openapi {
    title       = "SQL Server 2022 Members API"
    version     = "1.0.0"
    description = "Complete CRUD API backed by Microsoft SQL Server with stored procedure execution."
  }
}

connection "sqlserver" "main" {
  source = "sqlserver://sa:Password123!@127.0.0.1:1433?database=hclapi_db"

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
  description = "Lists all registered members from SQL Server."

  pipeline {
    sql "list" {
      connection = connection.sqlserver.main
      query      = "SELECT id, name, email, tier, points, created_at FROM members ORDER BY id DESC"
    }

    respond {
      status = 200
      body   = steps.list.rows
    }
  }
}

endpoint "POST /api/v1/members" {
  description = "Inserts member and catches SQL Server unique constraint codes (2627 and 2601)."

  request {
    body = schema.member_create
  }

  pipeline {
    sql "insert" {
      connection = connection.sqlserver.main
      query      = <<-SQL
        INSERT INTO members (name, email, tier)
        OUTPUT INSERTED.id, INSERTED.name, INSERTED.email, INSERTED.tier, INSERTED.points, INSERTED.created_at
        VALUES (@name, @email, @tier)
      SQL
      args = {
        name  = ctx.request.body.name
        email = ctx.request.body.email
        tier  = ctx.request.body.tier
      }

      # SQL Server Unique Constraint Violation Code (2627)
      catch "2627" {
        status = 409
        body   = problem(409, "Email address already registered", "email-collision")
      }

      # SQL Server Unique Index Duplicate Key Code (2601)
      catch "2601" {
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
  description = "Fetches a member by ID."

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
      connection = connection.sqlserver.main
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
  description = "Updates an existing member in SQL Server."

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
      connection = connection.sqlserver.main
      query      = <<-SQL
        UPDATE members
        SET
          name = COALESCE(@name, name),
          tier = COALESCE(@tier, tier)
        OUTPUT INSERTED.id, INSERTED.name, INSERTED.email, INSERTED.tier, INSERTED.points, INSERTED.created_at
        WHERE id = @id
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
      connection = connection.sqlserver.main
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
  description = "Executes a non-returning SQL Server stored procedure to award bonus points."

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
      connection = connection.sqlserver.main
      query      = "EXEC award_member_points @id, @bonus"
      args = {
        id    = ctx.request.path.id
        bonus = ctx.request.body.bonus
      }
    }

    respond {
      status = 200
      body = {
        status    = "Points awarded successfully"
        member_id = ctx.request.path.id
        awarded   = ctx.request.body.bonus
      }
    }
  }
}