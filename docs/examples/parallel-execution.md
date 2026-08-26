# Example 05: Parallel execution

Demonstrates concurrent DAG execution running three isolated SQL queries in parallel to aggregate user dashboard metrics into a single response.

## Manifest specification

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
}

endpoint "GET /api/v1/accounts/{id}/overview" {
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
        query      = "SELECT id, name, tier FROM accounts WHERE id = @id"
        args       = { id = ctx.request.path.id }
      }

      sql "fetch_invoices" {
        connection = connection.postgres.main
        query      = "SELECT id, amount_cents, status FROM invoices WHERE account_id = @id LIMIT 5"
        args       = { id = ctx.request.path.id }
      }

      sql "fetch_audit" {
        connection = connection.postgres.main
        query      = "SELECT id, action, created_at FROM audit_logs WHERE account_id = @id LIMIT 10"
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
```
