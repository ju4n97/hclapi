# Example 07: Modular production

Demonstrates a multi-file directory layout with primary/replica database connection pooling, global JWT authentication guards, and decoupled routes.

## Directory structure

```text
config/
├── tapir.hcl
├── connections.hcl
├── auth.hcl
├── schemas/
│   ├── user.hcl
│   └── order.hcl
└── routes/
    ├── users.hcl
    └── orders.hcl
```

## File specifications

### `config/connections.hcl`
```hcl
connection "postgres" "primary" {
  url = env("DATABASE_PRIMARY_URL")
  pool {
    max_open_conns = 50
  }
}

connection "postgres" "replica" {
  url = env("DATABASE_REPLICA_URL")
  pool {
    max_open_conns = 100
  }
}
```

### `config/auth.hcl`
```hcl
auth "jwt_bearer" {
  type   = "jwt"
  secret = env("JWT_SECRET")
  header = "Authorization"
  prefix = "Bearer "
}
```

### `config/tapir.hcl`
```hcl
api {
  prefix = "/api/v1"
  auth   = [auth.jwt_bearer]
}
```

### `config/routes/users.hcl`
```hcl
endpoint "GET /users/me" {
  pipeline {
    sql "find_user" {
      connection = connection.postgres.replica
      query      = "SELECT id, name, email FROM users WHERE id = @id"
      args = {
        id = ctx.auth.claims.sub
      }
    }

    respond {
      condition = steps.find_user.rows_affected == 0
      status    = 404
      body      = { error = "User identity not found" }
    }

    respond {
      status = 200
      body   = steps.find_user.result
    }
  }
}
```
