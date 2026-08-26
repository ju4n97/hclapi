# Transaction step

The `transaction` block groups multiple database operations into an atomic transaction.

## Transaction lifecycle

```mermaid
sequenceDiagram
    participant Pipeline as Engine Pipeline
    participant Tx as Dedicated Tx Handle (*sql.Tx)
    participant DB as PostgreSQL

    Pipeline->>DB: BEGIN Transaction
    Pipeline->>Tx: Step 1: INSERT user
    Tx-->>Pipeline: Returns ID
    Pipeline->>Tx: Step 2: INSERT workspace (with user ID)
    alt Any Step Errors
        Pipeline->>DB: ROLLBACK Transaction
        Pipeline-->>Pipeline: Terminate with HTTP 500 or Catch Status
    else All Steps Succeed
        Pipeline->>DB: COMMIT Transaction
    end
```

## Syntax

```hcl
transaction "provision_account" {
  connection = connection.postgres.main

  sql "create_user" {
    query = "INSERT INTO users (email) VALUES (@email) RETURNING id"
    args  = { email = ctx.request.body.email }
  }

  sql "create_profile" {
    query = "INSERT INTO profiles (user_id, name) VALUES (@uid, @name)"
    args = {
      uid  = steps.create_user.result.id
      name = ctx.request.body.name
    }
  }
}
```

## Rollback semantics

* All enclosed `sql` operations execute on a single dedicated transaction handle (`*sql.Tx`).
* If any enclosed step fails, or if an unhandled error occurs, the transaction automatically issues a `ROLLBACK`.
* Successful completion of the final child step triggers an explicit `COMMIT`.
