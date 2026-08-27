---
title: Transaction
description: Multi-table atomic database transactions with automated rollback.
---

# Transaction

The `transaction` block groups multiple database queries into an atomic transaction. If any query inside the block fails or triggers a constraint violation, the engine automatically rolls back all preceding operations in the transaction block and short-circuits pipeline execution.

## Block declaration

```hcl
transaction "<name>" {
  connection = connection.<driver>.<name>

  sql "step_one" {
    # ...
  }

  sql "step_two" {
    # ...
  }
}
```

## Attribute reference

| Attribute | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| **`name`** (Label) | `string` | Yes | Unique identifier for the atomic transaction block. |
| **`connection`** | `connection` | Yes | Database connection pool to acquire the transaction handle from. |
| **`sql`** (Blocks) | `block` | Yes | One or more sequential `sql` steps executed inside the transaction. |

## Chaining outputs within a transaction

Steps within a transaction block can reference the results of earlier steps executed within the same transaction scope:

```hcl
endpoint "POST /api/v1/onboard" {
  description = "Atomically registers a user and provisions their workspace"

  pipeline {
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          email = ctx.request.body.email.strip().lower()
          slug = email.split("@")[0]
          return {
            "email": email,
            "full_name": ctx.request.body.full_name.strip(),
            "slug": slug
          }
      STARLARK
    }

    # Atomic multi-table database transaction
    transaction "provision" {
      connection = connection.postgres.main

      # Statement 1: Insert user
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

        # Catch duplicate email constraint (PostgreSQL 23505)
        catch "23505" {
          abort_with_status = 409
          body = {
            error = "A user with this email address already exists"
          }
        }
      }

      # Statement 2: Insert workspace using user ID from Statement 1
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

        # Catch duplicate slug collision
        catch "23505" {
          abort_with_status = 409
          body = {
            error = "Workspace slug collision, please select a custom identifier"
          }
        }
      }
    }

    # Final response after successful COMMIT
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

## Rollback semantics

1. **Automatic rollback on query error**: If any query within the transaction block returns an error, the engine issues a `ROLLBACK` to the database driver immediately.
2. **Aborted catch blocks**: If a `catch` block triggers `abort_with_status`, the transaction rolls back, and the engine writes the specified error payload directly to the client.
3. **Automatic commit**: When all nested statements succeed without errors, the engine issues a `COMMIT` before continuing to downstream pipeline steps.
