# connection

Declares access parameters and pool configuration for a database or cache.

## Declaration

A `connection` block takes two labels: the driver and a unique name.

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

The connection is referenced elsewhere as `connection.<driver>.<name>`.

## Attributes

| Attribute | Type     | Required | Description                   |
| :-------- | :------- | :------- | :---------------------------- |
| `url`     | `string` | yes      | connection DSN or URI         |
| `pool`    | `block`  | no       | pool configuration, see below |

### pool

| Attribute           | Type       | Default | Description                                                |
| :------------------ | :--------- | :------ | :--------------------------------------------------------- |
| `max_open_conns`    | `int`      | `25`    | maximum open connections                                   |
| `max_idle_conns`    | `int`      | `5`     | maximum idle connections retained                          |
| `conn_max_lifetime` | `Duration` | `"30m"` | maximum time a connection is reused before being recreated |
| `idle_timeout`      | `Duration` | `"5m"`  | maximum idle time before eviction                          |
| `size`              | `int`      | `20`    | total pool capacity, used by cache drivers                 |

## Drivers

PostgreSQL, with distinct primary and replica pools.

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
    max_open_conns = 100
    max_idle_conns = 20
  }
}
```

SQLite, a single-writer, file-backed database.

```hcl
connection "sqlite" "main" {
  url = "file:/data/app.db?mode=rwc&_journal_mode=WAL"

  pool {
    max_open_conns = 1
    idle_timeout   = "10m"
  }
}
```

Redis.

```hcl
connection "redis" "cache" {
  url = env("REDIS_URL")

  pool {
    size         = 25
    idle_timeout = "5m"
  }
}
```

## Reference in a pipeline

```hcl
endpoint "GET /api/v1/users/{id}" {
  pipeline {
    sql "find_user" {
      connection = connection.postgres.replica
      query      = "SELECT id, name, email FROM users WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

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
