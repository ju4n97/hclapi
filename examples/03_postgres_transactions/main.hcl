server {
  host = "0.0.0.0"
  port = 8080
}

connection "postgres" "main" {
  url = env("DATABASE_URL")

  pool {
    max_open_conns    = 25
    max_idle_conns    = 5
    conn_max_lifetime = "30m"
  }
}

schema "onboard_request" {
  field "email" {
    type     = string
    required = true
    format   = "email"
  }
  field "full_name" {
    type     = string
    required = true
  }
}

endpoint "POST /api/v1/onboard" {
  description = "Atomically registers a user and provisions their personal workspace."

  request {
    body = schema.onboard_request
  }

  pipeline {
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          raw_email = ctx.request.body.email.strip().lower()
          name_parts = ctx.request.body.full_name.strip().split(" ")
          first_name = name_parts[0].lower() if len(name_parts) > 0 else "user"
          slug = first_name + "-" + raw_email.split("@")[0]
          
          return {
            "email": raw_email,
            "full_name": ctx.request.body.full_name.strip(),
            "slug": slug
          }
      STARLARK
    }

    transaction "provision" {
      connection = connection.postgres.main

      sql "insert_user" {
        query = <<-SQL
          INSERT INTO users (email, full_name)
          VALUES (@email, @full_name)
          RETURNING id, email, full_name, created_at
        SQL
        args = {
          email     = steps.normalize.result.email
          full_name = steps.normalize.result.full_name
        }

        catch "23505" {
          abort_with_status = 409
          body = {
            code  = "USER_ALREADY_EXISTS"
            error = "A user with this email address already exists"
          }
        }
      }

      sql "insert_workspace" {
        query = <<-SQL
          INSERT INTO workspaces (owner_id, slug, plan)
          VALUES (@owner_id, @slug, 'free')
          RETURNING id, slug, plan
        SQL
        args = {
          owner_id = steps.insert_user.result.id
          slug     = steps.normalize.result.slug
        }

        catch "23505" {
          abort_with_status = 409
          body = {
            code  = "SLUG_COLLISION"
            error = "Workspace slug collision, please select a custom identifier"
          }
        }
      }
    }

    respond {
      status = 201
      body = {
        user      = steps.insert_user.result
        workspace = steps.insert_workspace.result
      }
    }
  }
}