# Example 03: PostgreSQL transactions

Demonstrates multi-table atomic transactions with automatic rollback and direct error code mapping for PostgreSQL unique constraint violations (`23505`).

## Manifest specification

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
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
  request {
    body = schema.onboard_request
  }

  pipeline {
    transaction "provision" {
      connection = connection.postgres.main

      sql "insert_user" {
        query = <<-SQL
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
          body = { error = "Email address already registered" }
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
          slug     = ctx.request.body.email
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
```
