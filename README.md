```text
██╗  ██╗██╗   ██╗███╗   ███╗██╗
██║ ██╔╝██║   ██║████╗ ████║██║
█████╔╝ ██║   ██║██╔████╔██║██║
██╔═██╗ ██║   ██║██║╚██╔╝██║██║
██║  ██╗╚██████╔╝██║ ╚═╝ ██║██║
╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝╚═╝
```

A declarative API runtime that turns HCL manifests into structured HTTP services.

Instead of writing boilerplate handler functions, routing logic, and database mapping code, endpoints are defined in configuration files alongside their validation rules and execution pipelines. Hclapi parses these manifests at startup, statically checks references, and compiles them into a runnable service.

Hclapi runs standalone via its CLI or embeds directly into any Go application as a standard `http.Handler`.

## Use cases

- **Data APIs & microservices:** Expose REST endpoints directly over databases without an ORM or heavyweight backend framework.
- **Internal tools & ops endpoints:** Build authenticated admin APIs, maintenance hooks, and reporting tools with minimal overhead.
- **Prototypes & fast iteration:** Working APIs with real validation, transactions, and mocks before committing to custom application code.
- **Hybrid Go applications:** Mount Hclapi inside an existing Go codebase to handle repetitive CRUD routes while keeping complex business logic in native Go.

## Database & driver support

Hclapi connects to persistence layers through connection pools defined in HCL. Parameterized bindings are mapped automatically across supported engines:

| Driver                   | Identifier | Parameter Syntax                  |
| :----------------------- | :--------- | :-------------------------------- |
| **PostgreSQL**           | `postgres` | `@param` $\rightarrow$ `$1, $2`   |
| **SQLite**               | `sqlite`   | `@param` $\rightarrow$ `?`        |
| **MySQL / MariaDB**      | `mysql`    | `@param` $\rightarrow$ `?`        |
| **Microsoft SQL Server** | `mssql`    | `@param` $\rightarrow$ `@p1, @p2` |
| **Redis**                | `redis`    | Native command args               |

## Quick example

A complete endpoint handling input validation, Starlark normalization, parameterized PostgreSQL persistence, and constraint error mapping:

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")

  pool {
    max_open_conns = 25
    idle_timeout   = "5m"
  }
}

schema "user_create" {
  field "name" {
    type     = string
    required = true
  }
  field "email" {
    type     = string
    required = true
    format   = "email"
  }
}

endpoint "POST /api/v1/users" {
  description = "Registers a new user account."

  request {
    body = schema.user_create
  }

  pipeline {
    # 1. Normalize input in a sandboxed Starlark script
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
            body = ctx.request.body
            return {
                "name": body.name.strip(),
                "email": body.email.strip().lower()
            }
      STARLARK
    }

    # 2. Run parameterized SQL with explicit argument bindings
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

      # Map database constraint errors directly to HTTP responses
      catch "23505" {
        abort_with_status = 409
        body = { error = "Email is already registered" }
      }
    }

    # 3. Return the response
    respond {
      status = 201
      body   = steps.insert_user.result
    }
  }
}
```

## Embedding in Go

Hclapi mounts directly onto standard Go multiplexers and supports custom Go step functions for tasks requiring native SDKs, cryptography, or external service calls:

```go
package main

import (
 "log"
 "net/http"

 "github.com/ju4n97/hclapi"
)

func main() {
 // 1. Load and compile HCL manifests
 engine, err := hclapi.NewEngine(hclapi.Options{
    ManifestDir:  "./api",
    StrictTyping: true,
 })
 if err != nil {
    log.Fatalf("failed to load hclapi manifests: %v", err)
 }

 // 2. Register native Go functions callable inside HCL pipelines
 engine.RegisterStep("crypto.hash_token", func(ctx *hclapi.Context) (any, error) {
    rawToken := ctx.Args["token"].(string)
    return myHashFunction(rawToken), nil
 })

 // 3. Mount engine onto standard ServeMux
 mux := http.NewServeMux()
 mux.Handle("/api/", engine.Handler())

 // Native Go routes live alongside Hclapi routes without conflict
 mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
 })

 log.Println("Server running on :8080")
 log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Architecture & concepts

### Pipeline step reference

Endpoints execute as an ordered chain of steps. Steps share a request context (`ctx`) and pass data down the execution chain via `steps.<step_name>.result`.

| Step block    | Scope / Target     | Behavior                                                                                  |
| :------------ | :----------------- | :---------------------------------------------------------------------------------------- |
| `starlark`    | Isolated VM        | Executes side-effect-free data transformation scripts.                                    |
| `sql`         | Database pool      | Executes parameterized queries with strict HCL bindings.                                  |
| `redis`       | Redis pool         | Executes native cache lookups, counters, and key mutations with optional TTLs.            |
| `transaction` | Atomic DB block    | Wraps multiple SQL operations; rolls back automatically on error.                         |
| `parallel`    | Concurrent workers | Executes independent steps simultaneously and joins outputs into context.                 |
| `go`          | Host application   | Invokes Go functions registered via `RegisterStep`.                                       |
| `respond`     | HTTP writer        | **Terminal step.** Evaluates conditions, sets headers, writes status, and ends execution. |

### Starlark sandboxing

For payload normalization, filtering, or array operations, Hclapi embeds [Starlark](https://starlark-lang.org/). Starlark scripts are deterministic, cannot access the host network or filesystem, and run within pre-warmed VM pools to eliminate runtime memory allocations per request.

### SQL safety & named parameters

SQL injection is prevented at compile time. Queries require named parameters (`@field`), which are mapped to database driver arguments. Dynamic string interpolation (`${...}`) within SQL query strings is rejected during manifest validation.

### Authentication & route overrides

Authentication providers (such as JWT validators) can be declared globally and overridden per route:

```hcl
api {
  prefix = "/api/v1"
  auth   = [auth.jwt_main] # Applied globally
}

endpoint "POST /api/v1/public/webhook" {
  auth = [] # Explicitly public route override
}
```

### OpenAPI generation

Because endpoint paths, input schemas, and response shapes are declared in HCL, standard OpenAPI v3 specifications compile directly from active manifests:

```sh
hclapi openapi ./api > openapi.yaml
```

## Runnable examples

The [`examples/`](./examples) directory contains complete reference projects with Docker Compose configurations and seed data:

- **[`01_zero_dependency`](./examples/01_zero_dependency)**: In-memory mock data and Starlark payload transformations without an external database.
- **[`02_sqlite_crud`](./examples/02_sqlite_crud)**: Complete CRUD operations backed by a local SQLite file.
- **[`03_postgres_transactions`](./examples/03_postgres_transactions)**: Multi-table atomic transactions with constraint violation error handling.
- **[`04_redis_caching`](./examples/04_redis_caching)**: Cache-aside implementation using Redis and conditional early returns.
- **[`05_parallel_execution`](./examples/05_parallel_execution)**: Concurrent database queries executed in parallel and aggregated into a single dashboard response.
- **[`06_go_embedded`](./examples/06_go_embedded)**: Outbound HTTP requests and Go SDK integrations using native Go pipeline steps.
- **[`07_modular_production`](./examples/07_modular_production)**: Multi-file project layout with primary/replica connection pools and JWT authentication.

## CLI usage

Install the standalone binary:

```sh
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

Commands:

```sh
# Start serving endpoints from a manifest directory
hclapi serve --config ./api

# Validate manifest syntax, references, and schemas
hclapi validate ./api

# Compile active manifests into an OpenAPI v3 YAML file
hclapi openapi ./api > openapi.yaml
```

## License

[MIT](LICENSE)
