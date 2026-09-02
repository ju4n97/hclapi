> [!IMPORTANT]
> `hclapi` is in early development (`v0.1.x`) and follows documentation-driven development. [Some documented features haven't been implemented yet](https://github.com/ju4n97/hclapi/issues). Bugs and breaking changes are to be expected. Feedback and issue reports are welcome.

# hclapi

[![Go Reference](https://img.shields.io/badge/Go_Reference-pkg.go.dev-007D9C?style=flat-square)](https://pkg.go.dev/github.com/ju4n97/hclapi)
[![Release](https://img.shields.io/github/v/release/ju4n97/hclapi?style=flat-square&label=Release)](https://github.com/ju4n97/hclapi/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/ju4n97/hclapi/ci.yaml?style=flat-square&label=CI)](https://github.com/ju4n97/hclapi/actions/workflows/ci.yaml)

`hclapi` is a declarative backend runtime distributed as a single lightweight static binary. It compiles HashiCorp Configuration Language (HCL) manifests, SQL queries, and sandboxed Starlark scripts into structured HTTP services with native connection pooling, schema validation, and automatic OpenAPI 3.1 documentation.

Manifests are parsed and validated at boot time and executed directly at runtime. `hclapi` doesn't generate or compile Go code.

[Documentation](https://ju4n97.github.io/hclapi/) · [Quickstart](https://ju4n97.github.io/hclapi/docs/quickstart.html) · [Why hclapi](https://ju4n97.github.io/hclapi/docs/why.html) · [Patterns](https://ju4n97.github.io/hclapi/docs/patterns.html) · [Examples](./examples)

## Supported connectors

`hclapi` connects natively to databases and storage layers using zero-CGO pure Go drivers:

| Category              | Driver          | Supported engines                             | Status        |
| :-------------------- | :-------------- | :-------------------------------------------- | :------------ |
| **Relational SQL**    | `"postgres"`    | PostgreSQL, Supabase, TimescaleDB, AWS Aurora | `Available`   |
|                       | `"sqlite"`      | SQLite3, Turso, LibSQL                        | `Available`   |
|                       | `"mysql"`       | MySQL, MariaDB, PlanetScale, TiDB             | `Available`   |
|                       | `"sqlserver"`   | Microsoft SQL Server, Azure SQL               | `Available`   |
|                       | `"oracle"`      | Oracle Database 11g – 23ai                    | `Available`   |
|                       | `"cockroachdb"` | CockroachDB Dedicated & Serverless            | `Available`   |
| **Analytical SQL**    | `"clickhouse"`  | ClickHouse Cloud & Self-Hosted                | `Available`   |
|                       | `"duckdb"`      | DuckDB Embedded Columnar                      | `Available`   |
| **Key-Value / Cache** | `"redis"`       | Redis, Valkey, AWS ElastiCache                | `In-progress` |

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

## Quick install

### Linux and macOS

```bash
curl -fsSL https://raw.githubusercontent.com/ju4n97/hclapi/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/ju4n97/hclapi/main/scripts/install.ps1 | iex
```

### Container (Docker / Podman)

```bash
docker run --rm -p 8080:8080 -v "$(pwd):/app:ro" ghcr.io/ju4n97/hclapi:latest serve -c /app
```

### Using Go

```bash
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

_(Linux `.deb`, `.rpm`, `.apk`, and `.pkg.tar.zst` packages are available on the [releases page](https://github.com/ju4n97/hclapi/releases/latest))._

See the [Installation guide](https://ju4n97.github.io/hclapi/docs/installation.html) for package manager setup and verification.

## Embedding in Go

`hclapi.Engine` implements standard `http.Handler` and mounts directly into any Go HTTP router:

```go
package main

import (
  "log/slog"
  "net/http"
  "os"

  "github.com/ju4n97/hclapi"
)

func main() {
  logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

  engine, err := hclapi.NewEngine(hclapi.Options{
    ConfigPath:   "./api",
    StrictTyping: true,
    Logger:       logger,
  })
  if err != nil {
    logger.Error("engine initialization failed", "error", err)
    os.Exit(1)
  }
  defer engine.Close()

  mux := http.NewServeMux()
  mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("OK"))
  })
  mux.Handle("/", engine.Handler())

  logger.Info("server listening on :8080")
  _ = http.ListenAndServe(":8080", mux)
}
```

See [Go integration](https://ju4n97.github.io/hclapi/guides/go.html) for custom error handlers, logging, and registering native Go steps.

## Documentation

Full reference documentation covering the request lifecycle, manifest block syntax, and patterns is available at: [ju4n97.github.io/hclapi](https://ju4n97.github.io/hclapi/)

- [Request lifecycle](https://ju4n97.github.io/hclapi/docs/concepts/lifecycle.html)
- [Execution context](https://ju4n97.github.io/hclapi/docs/concepts/context.html)
- [Pipelines and steps](https://ju4n97.github.io/hclapi/docs/concepts/pipelines.html)
- [Manifest structure and merging](https://ju4n97.github.io/hclapi/docs/manifest/structure.html)
- [OpenAPI configuration](https://ju4n97.github.io/hclapi/openapi/overview.html)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for architectural guidelines, repository structure, and development workflows.

## License

[MIT](LICENSE)
