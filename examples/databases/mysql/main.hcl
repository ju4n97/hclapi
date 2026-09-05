server {
  host = "127.0.0.1"
  port = 8080

  openapi {
    title       = "MySQL 8 Members API"
    version     = "1.0.0"
    description = "Complete CRUD API backed by MySQL 8 with stored procedure execution."
  }
}

connection "mysql" "main" {
  source = "hclapi:hclapi_password@tcp(127.0.0.1:3306)/hclapi_db?parseTime=true"

  pool {
    max_open = 30
    max_idle = 5
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
  description = "Lists all members from MySQL."

  pipeline {
    sql "list" {
      connection = connection.mysql.main
      query      = "SELECT id, name, email, tier, points, created_at FROM members ORDER BY id DESC"
    }

    respond {
      status = 200
      body   = steps.list.rows
    }
  }
}

endpoint "POST /api/v1/members" {
  description = "Inserts member into MySQL and catches duplicate key code (1062)."

  request {
    body = schema.member_create
  }

  pipeline {
    sql "insert" {
      connection = connection.mysql.main
      query      = "INSERT INTO members (name, email, tier) VALUES (@name, @email, @tier)"
      args = {
        name  = ctx.request.body.name
        email = ctx.request.body.email
        tier  = ctx.request.body.tier
      }

      catch "1062" {
        status = 409
        body   = problem(409, "Email address already registered", "email-collision")
      }
    }

    respond {
      status = 201
      body   = { message = "Member created successfully", affected = steps.insert.rows_affected }
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
      connection = connection.mysql.main
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
      connection = connection.mysql.main
      query      = <<-SQL
        UPDATE members
        SET
          name = COALESCE(@name, name),
          tier = COALESCE(@tier, tier)
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
      body   = { message = "Member updated", affected = steps.update.rows_affected }
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
      connection = connection.mysql.main
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
  description = "Executes a MySQL stored procedure to award bonus points."

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
    sql "call_procedure" {
      connection = connection.mysql.main
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