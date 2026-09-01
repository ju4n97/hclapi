# hclapi

[![Go Reference](https://pkg.go.dev/badge/github.com/ju4n97/hclapi.svg?style=flat)](https://pkg.go.dev/github.com/ju4n97/hclapi)
[![Release](https://github.com/ju4n97/hclapi/actions/workflows/release.yaml/badge.svg?style=flat)](https://github.com/ju4n97/hclapi/actions/workflows/release.yaml)
[![CI](https://github.com/ju4n97/hclapi/actions/workflows/ci.yaml/badge.svg?style=flat)](https://github.com/ju4n97/hclapi/actions/workflows/ci.yaml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`hclapi` is a declarative backend runtime distributed as a single static binary. It compiles HashiCorp Configuration Language (HCL) manifests, SQL queries, and sandboxed Starlark scripts into production HTTP APIs with native connection pooling, schema validation, and OpenAPI documentation.

Manifests are parsed and executed at runtime. `hclapi` does not generate or compile Go code.

[Documentation](https://ju4n97.github.io/hclapi/) ·
[Why hclapi](https://ju4n97.github.io/hclapi/why.html) ·
[Patterns](https://ju4n97.github.io/hclapi/patterns.html) ·
[Examples](./examples)

## Supported connectors

`hclapi` connects to any data source using native connection pooling and zero-CGO pure Go drivers:

| Category              | Driver          | Supported engines                             |
| :-------------------- | :-------------- | :-------------------------------------------- |
| **Relational SQL**    | `"postgres"`    | PostgreSQL, Supabase, TimescaleDB, AWS Aurora |
|                       | `"sqlite"`      | SQLite3, Turso, LibSQL                        |
|                       | `"mysql"`       | MySQL, MariaDB, PlanetScale, TiDB             |
|                       | `"sqlserver"`   | Microsoft SQL Server, Azure SQL               |
|                       | `"oracle"`      | Oracle Database 11g – 23ai                    |
|                       | `"cockroachdb"` | CockroachDB Dedicated & Serverless            |
| **Analytical SQL**    | `"clickhouse"`  | ClickHouse Cloud & Self-Hosted                |
|                       | `"duckdb"`      | DuckDB Embedded Columnar                      |
| **Key-Value / Cache** | `"redis"`       | Redis, Valkey, AWS ElastiCache                |
| **Blob Storage**      | `"s3"`          | Amazon S3, Cloudflare R2, MinIO, GCS          |

## Example

A production user registration endpoint with input normalization, parameterized SQL insertion, constraint collision interception, and structured RFC 9457 error responses:

```hcl
server {
  host          = "0.0.0.0"
  port          = 8080
  max_body_size = "5MB"
}

connection "postgres" "main" {
  url = env("DATABASE_URL")

  pool {
    max_open_conns    = 25
    conn_max_lifetime = "30m"
  }
}

schema "user_create" {
  field "email" {
    type        = string
    required    = true
    format      = "email"
    description = "Primary user login and notification email"
  }

  field "full_name" {
    type       = string
    required   = true
    min_length = 2
    max_length = 100
  }

  field "role" {
    type    = string
    default = "member"
    enum    = ["admin", "member", "viewer"]
  }
}

endpoint "POST /api/v1/users" {
  description = "Registers a new user account and provisions a default workspace."

  request {
    body = schema.user_create
  }

  pipeline {
    # 1. Sandboxed data transformation
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          email = ctx.request.body.get("email", "").strip().lower()
          name = ctx.request.body.get("full_name", "").strip()
          return {
            "email": email,
            "name": name,
            "handle": email.split("@")[0]
          }
      STARLARK
    }

    # 2. Parameterized SQL insert with constraint interception
    sql "insert_user" {
      connection = connection.postgres.main
      query      = <<-SQL
        INSERT INTO users (email, name, role)
        VALUES (@email, @name, @role)
        RETURNING id, email, name, role, created_at
      SQL
      args = {
        email = steps.normalize.result.email
        name  = steps.normalize.result.name
        role  = ctx.request.body.role
      }

      # Intercept PostgreSQL unique violation (code 23505)
      catch "23505" {
        status  = 409
        headers = { 
          "X-Error" = "Conflict" 
        }
        body    = problem(409, "A user with this email address already exists", "email-collision")
      }
    }

    # 3. Terminal 201 response with created record
    respond {
      status  = 201
      headers = { 
        "Location" = "/api/v1/users/${steps.insert_user.row.id}" 
      }
      body    = steps.insert_user.row
    }
  }
}
```

## Installation

Download precompiled binaries from the [releases page](https://github.com/ju4n97/hclapi/releases) or install via Go:

```sh
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

```sh
hclapi serve -c ./api
```

See [Installation](https://ju4n97.github.io/hclapi/installation.html) for detailed setup and [Quickstart](https://ju4n97.github.io/hclapi/quickstart.html) for a 5-minute tutorial.

## Embedding in Go

`hclapi` implements the standard `http.Handler` interface and mounts directly into any Go HTTP router:

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
    log.Fatalf("failed to initialize hclapi: %v", err)
  }
  defer engine.Close()

  mux := http.NewServeMux()

  mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
  })

  mux.Handle("/api/v1/", engine.Handler())

  log.Println("Server running on :8080")
  http.ListenAndServe(":8080", mux)
}
```

See [Go integration](https://ju4n97.github.io/hclapi/go/README.html) for custom error handlers, logging, and registering native Go steps.

## Documentation

Full reference documentation covering the request lifecycle, manifest block syntax, and patterns is available at: <https://ju4n97.github.io/hclapi/>

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for architecture details, package layouts, and local development instructions.

## License

[MIT](LICENSE)
