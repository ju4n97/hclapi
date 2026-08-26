```text
██╗  ██╗██╗   ██╗███╗   ███╗██╗
██║ ██╔╝██║   ██║████╗ ████║██║
█████╔╝ ██║   ██║██╔████╔██║██║
██╔═██╗ ██║   ██║██║╚██╔╝██║██║
██║  ██╗╚██████╔╝██║ ╚═╝ ██║██║
╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝╚═╝
```

Declarative HTTP API engine that compiles HCL manifests into standalone, production-ready services with parameterized SQL, sandboxed Starlark, and Redis caching.

[Documentation](https://ju4n97.github.io/hclapi/) · [Examples](./examples) · [Manifest syntax](https://ju4n97.github.io/hclapi/manifest/configuration.md)

## Overview

Hclapi is a language-agnostic API engine. It allows building and exposing structured HTTP endpoints directly over databases and data workflows without needing a full backend framework.

- Configured entirely via HCL, Starlark, and SQL. Distributed as a single, zero-dependency cross-platform binary or Docker image.
- Deterministic execution order with strictly typed variable references.
- Sandboxed Starlark transformations with zero host side effects.
- Compile-time SQL safety via parameterized query enforcement (\`@param\`) with zero dynamic string interpolation.
- Optional Go embedding: Implements the standard library \`http.Handler\` interface for people that want to embed the engine in existing Go codebases.

## Quick example

A complete standalone API endpoint handling input validation, Starlark normalization, parameterized PostgreSQL persistence, constraint error mapping and OpenAPI generation in 3 steps:

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
}

schema "user_create" {
  field "name"  { type = string, required = true }
  field "email" { type = string, required = true, format = "email" }
}

endpoint "POST /api/v1/users" {
  description = "Registers a new user account."

  request {
    body = schema.user_create
  }

  pipeline {
    # 1. Normalize input in sandboxed Starlark
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          return {
            "name": ctx.request.body.name.strip(),
            "email": ctx.request.body.email.strip().lower()
          }
      STARLARK
    }

    # 2. Parameterized SQL persistence
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
        abort_with_status = 409
        body = { error = "Email address already registered" }
      }
    }

    # 3. Terminal response
    respond {
      status = 201
      body   = steps.insert_user.result
    }
  }
}
```

## Running standalone

Hclapi runs as a zero-dependency binary across Linux, macOS, and Windows:

Install the binary:

```sh
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

Usage:

```sh
# Start daemon from manifest
hclapi serve --config ./Hclapifile

# Validate syntax and references
hclapi validate ./api

# Compile OpenAPI v3 specification
hclapi openapi ./api > openapi.yaml
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
  engine, err := hclapi.NewEngine(hclapi.Options{
    ManifestDir:  "./api",
    StrictTyping: true,
  })
  if err != nil {
    log.Fatalf("engine initialization failed: %v", err)
  }

  mux := http.NewServeMux()
  mux.Handle("/api/v1/", engine.Handler())

  log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Documentation

Full architectural guides, manifest references, and step manuals are available on the [documentation site](https://ju4n97.github.io/hclapi/).

- [Getting started](https://ju4n97.github.io/hclapi/guide/getting-started)
- [Pipeline state machine](https://ju4n97.github.io/hclapi/guide/state-machine)
- [Manifest syntax](https://ju4n97.github.io/hclapi/manifest/configuration)
- [Pipeline steps](https://ju4n97.github.io/hclapi/steps/starlark)
- [Examples catalog](https://ju4n97.github.io/hclapi/examples/overview)

## License

[MIT](LICENSE)
