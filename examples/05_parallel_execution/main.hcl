server {
  host = "0.0.0.0"
  port = 8080
}

connection "postgres" "main" {
  url = env("DATABASE_URL")
}

endpoint "GET /api/v1/accounts/{id}/overview" {
  description = "Aggregates account details, invoices, and audit events concurrently."

  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }
  }

  pipeline {
    parallel {
      sql "fetch_account" {
        connection = connection.postgres.main
        query      = <<-SQL
          SELECT id, name, tier
          FROM accounts
          WHERE id = @id
        SQL
        args       = { id = ctx.request.path.id }
      }

      sql "fetch_invoices" {
        connection = connection.postgres.main
        query      = <<-SQL
          SELECT id, amount_cents, status, issued_at
          FROM invoices
          WHERE account_id = @id
          ORDER BY issued_at DESC
          LIMIT 5
        SQL
        args       = { id = ctx.request.path.id }
      }

      sql "fetch_audit" {
        connection = connection.postgres.main
        query      = <<-SQL
          SELECT id, action, created_at
          FROM audit_logs
          WHERE account_id = @id
          ORDER BY created_at DESC
          LIMIT 10
        SQL
        args       = { id = ctx.request.path.id }
      }
    }

    respond {
      condition = steps.fetch_account.rows_affected == 0
      status    = 404
      body      = { error = "Account not found" }
    }

    respond {
      status = 200
      body = {
        account         = steps.fetch_account.result
        recent_invoices = steps.fetch_invoices.result
        recent_events   = steps.fetch_audit.result
      }
    }
  }
}