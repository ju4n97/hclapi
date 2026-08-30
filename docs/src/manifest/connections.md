# connection

Declares access parameters, connection pool sizing, and lifecycle policies for a database, cache, or storage backend.

## Declaration

A `connection` block takes two labels: the canonical driver identifier and a unique connection name.

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

The connection is referenced in pipeline steps as `connection.<driver>.<name>`.

## Attributes

| Attribute    | Type     | Required | Description                                                                |
| :----------- | :------- | :------- | :------------------------------------------------------------------------- |
| driver label | `string` | yes      | Canonical driver identifier (e.g. `"postgres"`, `"sqlite"`, `"sqlserver"`) |
| name label   | `string` | yes      | Unique name identifier within the driver namespace                         |
| `url`        | `string` | yes      | Connection DSN or URI (supports `env(...)` resolution)                     |
| `pool`       | `block`  | no       | Connection pool tuning parameters                                          |

### pool

| Attribute           | Type       | Default | Description                                                  |
| :------------------ | :--------- | :------ | :----------------------------------------------------------- |
| `max_open_conns`    | `int`      | `25`    | Maximum number of open connections in the pool               |
| `max_idle_conns`    | `int`      | `5`     | Maximum number of idle connections retained                  |
| `conn_max_lifetime` | `Duration` | `"30m"` | Maximum time a connection may be reused before recreation    |
| `idle_timeout`      | `Duration` | `"5m"`  | Maximum idle duration before an unused connection is evicted |
| `size`              | `int`      | `20`    | Pool capacity used by cache and key-value drivers            |

## Supported drivers matrix

hclapi uses strictly canonical driver names across all supported relational, analytical, cache, and storage engines:

| Canonical driver    | Category          | Parameter placeholder | Supported databases and cloud platforms                   |
| :------------------ | :---------------- | :-------------------: | :-------------------------------------------------------- |
| **`"postgres"`**    | Relational SQL    |       `$1, $2`        | PostgreSQL, Supabase, TimescaleDB, Amazon Aurora Postgres |
| **`"sqlite"`**      | Embedded SQL      |          `?`          | SQLite3, Turso, LibSQL                                    |
| **`"mysql"`**       | Relational SQL    |          `?`          | MySQL, MariaDB, PlanetScale, TiDB, Amazon Aurora MySQL    |
| **`"sqlserver"`**   | Relational SQL    |      `@p1, @p2`       | Microsoft SQL Server, Azure SQL Database                  |
| **`"oracle"`**      | Relational SQL    |       `:1, :2`        | Oracle Database 11g, 12c, 19c, 21c, 23ai                  |
| **`"cockroachdb"`** | Distributed SQL   |       `$1, $2`        | CockroachDB Dedicated and Serverless                      |
| **`"clickhouse"`**  | Columnar SQL      |          `?`          | ClickHouse Cloud, ClickHouse Self-Hosted                  |
| **`"duckdb"`**      | Embedded Columnar |          `?`          | DuckDB Analytics Engine                                   |
| **`"redis"`**       | Key-Value / Cache |          N/A          | Redis, Valkey, AWS ElastiCache                            |
| **`"s3"`**          | Blob Storage      |          N/A          | Amazon S3, Cloudflare R2, MinIO, Google Cloud Storage     |

## Examples

### PostgreSQL

Configured with dedicated primary and read-replica connection pools:

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

### SQLite

File-backed database running in WAL mode with single-writer serialization:

```hcl
connection "sqlite" "main" {
  url = "file:/data/app.db?mode=rwc&_journal_mode=WAL"

  pool {
    max_open_conns = 1
    idle_timeout   = "10m"
  }
}
```

### MySQL

```hcl
connection "mysql" "main" {
  url = env("MYSQL_URL") # format: user:pass@tcp(localhost:3306)/dbname?parseTime=true

  pool {
    max_open_conns = 30
    max_idle_conns = 5
  }
}
```

### Microsoft SQL Server

```hcl
connection "sqlserver" "main" {
  url = env("SQLSERVER_URL") # format: sqlserver://user:pass@localhost:1433?database=appdb

  pool {
    max_open_conns = 25
    conn_max_lifetime = "30m"
  }
}
```

### Oracle Database

```hcl
connection "oracle" "enterprise" {
  url = env("ORACLE_URL") # format: oracle://user:pass@localhost:1521/service_name

  pool {
    max_open_conns = 20
    idle_timeout   = "5m"
  }
}
```

### ClickHouse

```hcl
connection "clickhouse" "analytics" {
  url = env("CLICKHOUSE_URL") # format: clickhouse://user:pass@localhost:9000/analytics

  pool {
    max_open_conns = 10
  }
}
```

### Redis / Valkey

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

Step blocks reference connections by their two-part identifier (`connection.<driver>.<name>`):

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
      value      = json_encode(steps.find_user.row)
      ttl        = "15m"
    }

    respond {
      status = 200
      body   = steps.find_user.row
    }
  }
}
```
