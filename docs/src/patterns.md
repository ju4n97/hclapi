# Patterns

Recurring pipeline patterns referenced across endpoint configurations.

## 404 on a missing record

Query a single resource, verify `rows_affected`, and return `steps.<name>.row`.

```hcl
endpoint "GET /api/v1/users/{id}" {
  pipeline {
    sql "find_user" {
      connection = connection.postgres.main
      query      = "SELECT id, name, email FROM users WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    respond {
      condition = steps.find_user.rows_affected == 0
      status    = 404
      body      = { error = "User not found" }
    }

    respond {
      status = 200
      body   = steps.find_user.row
    }
  }
}
```

## Cache aside

Read the cache, respond immediately on a hit, fall back to the database on a miss, and store the result back into cache.

```hcl
endpoint "GET /api/v1/products/{sku}" {
  pipeline {
    redis "cache_lookup" {
      connection = connection.redis.cache
      command    = "GET"
      key        = "cache:product:${ctx.request.path.sku}"
    }

    respond {
      condition = steps.cache_lookup.value != null
      status    = 200
      headers   = { "X-Cache" = "HIT" }
      body      = json_decode(steps.cache_lookup.value)
    }

    sql "db_query" {
      connection = connection.postgres.main
      query      = "SELECT id, sku, name, price_cents, inventory FROM products WHERE sku = @sku"
      args       = { sku = ctx.request.path.sku }
    }

    respond {
      condition = steps.db_query.rows_affected == 0
      status    = 404
      body      = { error = "Product not found" }
    }

    redis "cache_write" {
      connection = connection.redis.cache
      command    = "SET"
      key        = "cache:product:${ctx.request.path.sku}"
      value      = json_encode(steps.db_query.row)
      ttl        = "30m"
    }

    respond {
      status  = 200
      headers = { "X-Cache" = "MISS" }
      body    = steps.db_query.row
    }
  }
}
```

## Transactional writes

Two inserts executed atomically inside a transaction, catching constraint collisions.

```hcl
endpoint "POST /api/v1/onboard" {
  pipeline {
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          email = ctx.request.body.get("email", "").strip().lower()
          return {
            "email": email,
            "full_name": ctx.request.body.get("full_name", "").strip(),
            "slug": email.split("@")[0]
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
          status = 409
          body   = { error = "A user with this email address already exists" }
        }
      }

      sql "insert_workspace" {
        query = <<-SQL
          INSERT INTO workspaces (owner_id, slug, plan)
          VALUES (@owner_id, @slug, 'free')
          RETURNING id, slug, plan
        SQL
        args = {
          owner_id = steps.insert_user.row.id
          slug     = steps.normalize.result.slug
        }

        catch "23505" {
          status = 409
          body   = { error = "Workspace slug collision, please select a custom identifier" }
        }
      }
    }

    respond {
      status = 201
      body = {
        user      = steps.insert_user.row
        workspace = steps.insert_workspace.row
      }
    }
  }
}
```

## Parallel aggregation

Multiple independent queries executed concurrently and joined into one response payload.

```hcl
endpoint "GET /api/v1/accounts/{id}/overview" {
  pipeline {
    parallel {
      sql "fetch_account" {
        connection = connection.postgres.main
        query      = "SELECT id, name, tier FROM accounts WHERE id = @id"
        args       = { id = ctx.request.path.id }
      }

      sql "fetch_invoices" {
        connection = connection.postgres.main
        query      = <<-SQL
          SELECT id, amount_cents, status, issued_at
          FROM invoices WHERE account_id = @id
          ORDER BY issued_at DESC LIMIT 5
        SQL
        args = { id = ctx.request.path.id }
      }

      sql "fetch_audit" {
        connection = connection.postgres.main
        query      = <<-SQL
          SELECT id, action, created_at
          FROM audit_logs WHERE account_id = @id
          ORDER BY created_at DESC LIMIT 10
        SQL
        args = { id = ctx.request.path.id }
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
        account         = steps.fetch_account.row
        recent_invoices = steps.fetch_invoices.rows
        recent_events   = steps.fetch_audit.rows
      }
    }
  }
}
```
