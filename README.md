# hclapi

[![Go Reference](https://pkg.go.dev/badge/github.com/ju4n97/hclapi.svg?style=flat)](https://pkg.go.dev/github.com/ju4n97/hclapi)
[![Release](https://github.com/ju4n97/hclapi/actions/workflows/release.yaml/badge.svg?style=flat)](https://github.com/ju4n97/hclapi/actions/workflows/release.yaml)
[![CI](https://github.com/ju4n97/hclapi/actions/workflows/ci.yaml/badge.svg?style=flat)](https://github.com/ju4n97/hclapi/actions/workflows/ci.yaml)

hclapi is a backend engine distributed as a single binary. It turns HashiCorp Configuration Language (HCL) manifests into HTTP APIs,
combining data access, business logic, validation, and API definitions in a single declarative configuration, with built-in OpenAPI
generation.

[Documentation](https://ju4n97.github.io/hclapi/) ·
[Why hclapi](https://ju4n97.github.io/hclapi/why.html) ·
[Examples](./examples)

## Supported connectors

hclapi connects to any data source using native connection pooling and zero-CGO pure Go drivers:

| Category              | Driver          | Supported engines                             |
| :-------------------- | :-------------- | :-------------------------------------------- |
| **Relational SQL**    | `"postgres"`    | PostgreSQL, Supabase, TimescaleDB, AWS Aurora |
|                       | `"sqlite"`      | SQLite3, Turso, LibSQL                        |
|                       | `"mysql"`       | MySQL, MariaDB, PlanetScale, TiDB             |
|                       | `"sqlserver"`   | Microsoft SQL Server, Azure SQL               |
|                       | `"oracle"`      | Oracle Database 11g – 23ai                    |
|                       | `"cockroachdb"` | CockroachDB                                   |
| **Analytical SQL**    | `"clickhouse"`  | ClickHouse Cloud & Self-Hosted                |
|                       | `"duckdb"`      | DuckDB Embedded Columnar                      |
| **Key-Value / Cache** | `"redis"`       | Redis, Valkey, AWS ElastiCache                |
| **Blob Storage**      | `"s3"`          | Amazon S3, Cloudflare R2, MinIO, GCS          |

## Example

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
}

schema "user_create" {
  field "name"  { type = string, required = true }
  field "email" { type = string, required = true, format = "email" }
}

endpoint "POST /api/v1/users" {
  request {
    body = schema.user_create
  }

  pipeline {
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          return {
            "name": ctx.request.body.name.strip(),
            "email": ctx.request.body.email.strip().lower()
          }
      STARLARK
    }

    sql "insert_user" {
      connection = connection.postgres.main
      query      = <<-SQL
        INSERT INTO users (name, email)
        VALUES (@name, @email)
        RETURNING id, name, email, created_at
      SQL
      args = {
        name  = steps.normalize.result.name
        email = steps.normalize.result.email
      }

      catch "23505" {
        status = 409
        body   = { error = "Email address already registered" }
      }
    }

    respond {
      status = 201
      body   = steps.insert_user.row
    }
  }
}
```

## Installation

```sh
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

```sh
hclapi serve -c ./api
```

See [Installation](https://ju4n97.github.io/hclapi/installation.html) for precompiled cross-platform binaries, and [Quickstart](https://ju4n97.github.io/hclapi/quickstart.html) for a complete walkthrough.

## Embedding in Go

hclapi implements the standard library `http.Handler` interface and mounts onto any Go HTTP router:

```go
package main

import (
 "log"
 "net/http"

 "github.com/ju4n97/hclapi"
)

func main() {
  engine, err := hclapi.NewEngine(hclapi.Options{
    ConfigPath:   "./api",
    StrictTyping: true,
  })
  if err != nil {
    log.Fatal(err)
  }

  mux := http.NewServeMux()
  mux.Handle("/api/v1/", engine.Handler())

  log.Println("Server running on :8080")
  http.ListenAndServe(":8080", mux)
}
```

See [Go integration](https://ju4n97.github.io/hclapi/go/README.html) for custom error handlers, structured logging, and registering native Go steps.

## Documentation

Full reference for the request lifecycle, manifest syntax, pipeline steps, and patterns is available at <https://ju4n97.github.io/hclapi/>.

## License

[MIT](LICENSE)
