---
title: Connections & pooling
description: Declaring connection blocks, driver types, credentials via environment variables, and connection pool configurations.
---

# Connections & pooling

The `connection` block defines access parameters, credentials, and connection pool configurations for external data persistence layers, relational databases, document stores, and in-memory caches.

## Block declaration

A `connection` block requires two labels: the **driver type** and a **unique identifier**:

```hcl
connection "<driver>" "<name>" {
  url = env("DATABASE_URL")

  pool {
    max_open_conns    = 25
    max_idle_conns    = 5
    conn_max_lifetime = "30m"
    idle_timeout      = "5m"
  }
}
```

Once declared, the connection is globally referenceable across any pipeline in the manifest tree using `connection.<driver>.<name>`.

## Attribute reference

| Attribute  | Type     | Required | Description                                                                   |
| :--------- | :------- | :------- | :---------------------------------------------------------------------------- |
| **`url`**  | `string` | Yes      | Connection DSN or URI.                                                        |
| **`pool`** | `block`  | No       | Connection pool tuning sub-block. Applies driver-specific pooling parameters. |

### Pool configuration attributes (`pool`)

| Attribute               | Type       | Default | Description                                                                    |
| :---------------------- | :--------- | :------ | :----------------------------------------------------------------------------- |
| **`max_open_conns`**    | `int`      | `25`    | Maximum number of open connections established to the database.                |
| **`max_idle_conns`**    | `int`      | `5`     | Maximum number of idle connections retained in the pool.                       |
| **`conn_max_lifetime`** | `Duration` | `"30m"` | Maximum duration a connection may be reused before being closed and recreated. |
| **`idle_timeout`**      | `Duration` | `"5m"`  | Maximum duration an idle connection remains in the pool before eviction.       |
| **`size`**              | `int`      | `20`    | Total client pool capacity (commonly used for key-value and cache drivers).    |

## Driver configurations

### 1. PostgreSQL connection

Configures connection pooling with distinct primary (write) and replica (read) pool limits:

```hcl
connection "postgres" "primary" {
  url = env("DATABASE_PRIMARY_URL")

  pool {
    max_open_conns    = 50
    max_idle_conns    = 10
    conn_max_lifetime = "1h"
    idle_timeout      = "10m"
  }
}

connection "postgres" "replica" {
  url = env("DATABASE_REPLICA_URL")

  pool {
    max_open_conns    = 100
    max_idle_conns    = 20
    conn_max_lifetime = "1h"
    idle_timeout      = "10m"
  }
}
```

### 2. SQLite embedded database

Configures a single-writer file-based database connection using SQLite connection string URI parameters:

```hcl
connection "sqlite" "main" {
  url = "file:/data/app.db?mode=rwc&_journal_mode=WAL"

  pool {
    max_open_conns = 1
    idle_timeout   = "10m"
  }
}
```

### 3. Redis and key-value cache

Configures an in-memory cache connection pool:

```hcl
connection "redis" "cache" {
  url = env("REDIS_URL")

  pool {
    size         = 25
    idle_timeout = "5m"
  }
}
```

## Referencing connections in pipeline steps

Steps bind to connection pools by passing the identifier to their `connection` attribute:

```hcl
endpoint "GET /api/v1/users/{id}" {
  pipeline {
    # 1. Read from PostgreSQL replica pool
    sql "find_user" {
      connection = connection.postgres.replica
      query      = "SELECT id, name, email FROM users WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    # 2. Write to Redis cache connection
    redis "cache_user" {
      connection = connection.redis.cache
      command    = "SET"
      key        = "user:${ctx.request.path.id}"
      value      = json_encode(steps.find_user.result)
      ttl        = "15m"
    }

    respond {
      status = 200
      body   = steps.find_user.result
    }
  }
}
```
