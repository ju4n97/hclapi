---
title: Files & tree merging
description: File discovery, extension conventions, directory walking, and AST merging rules.
---

# Files & tree merging

Hclapi builds its runtime Abstract Syntax Tree (AST) by parsing either a single file or by walking a directory tree to merge multiple manifest files into a unified service definition.

## Recognized file formats

The parser identifies manifest files based on filename and extension:

| Pattern | Description | Example |
| :--- | :--- | :--- |
| **`Hclapifile`** | Extensionless manifest file. Primary convention for project roots. | `Hclapifile`, `routes/v1/Hclapifile` |
| **`*.hcl`** | Standard HashiCorp Configuration Language files. | `main.hcl`, `connections.hcl`, `users.hcl` |
| **`*.hclapi`** | Specialized Hclapi manifest extension. | `api.hclapi`, `orders.hclapi` |

Non-manifest files (such as `README.md`, `init.sql`, `.gitignore`, or static assets) are ignored during directory discovery.

## Directory scanning rules

When passing a directory path to the engine (e.g. `hclapi serve -c ./config`), the parser recursively walks the filesystem tree using the following rules:

1. **Recursive traversal**: Subdirectories are traversed to any depth, allowing routes and configurations to be organized logically by domain or API version.
2. **Hidden directory exclusion**: Any directory starting with a dot (such as `.git`, `.github`, or `.cache`) is skipped entirely, preventing corrupted or temporary files from being evaluated.
3. **Additive aggregation**: All declared endpoints, connections, schemas, and server configurations across all discovered files are merged into a single AST.

## AST merging semantics

When multiple files are merged into a unified manifest, the engine applies specific collision and precedence rules:

### Endpoint uniqueness

Endpoint definitions are identified by the combination of their HTTP method and URI pattern (e.g. `POST /api/v1/users`). If the same HTTP method and path combination is declared across multiple files, the parser halts at startup with a diagnostic error detailing the conflicting file paths.

### Server configuration precedence

If multiple `server { ... }` blocks are declared across separate files in the directory tree, the last evaluated block overwrites previous values for any explicitly declared attributes. Unset attributes retain baseline defaults.

### Global identifier scope

Declared block labels exist in a global namespace across the entire merged tree:

* A connection labeled `connection "postgres" "primary"` defined in `connections.hcl` is directly accessible by all endpoint pipelines in `routes/users.hcl` and `routes/orders.hcl` as `connection.postgres.primary`.
* A schema labeled `schema "user_create"` defined in `schemas/user.hcl` can be referenced by any endpoint in the project as `schema.user_create`.

## Project layouts

### 1. Flat layout (Single-file services)

Suitable for simple microservices, webhooks, or proof-of-concept endpoints:

```text
my-service/
├── Hclapifile
└── docker-compose.yaml
```

**`Hclapifile`:**

```hcl
server {
  host = "0.0.0.0"
  port = 8080
}

connection "postgres" "main" {
  url = env("DATABASE_URL")
}

endpoint "GET /health" {
  pipeline {
    respond {
      status = 200
      body   = { status = "OK" }
    }
  }
}
```

### 2. Domain-driven modular layout

Recommended for production applications with multiple resources, shared connections, and dedicated schema definitions:

```text
api-service/
├── server.hcl
├── connections.hcl
├── schemas/
│   ├── account.hcl
│   └── user.hcl
└── routes/
    ├── accounts.hcl
    └── users.hcl
```

**`connections.hcl`:**

```hcl
connection "postgres" "primary" {
  url = env("DATABASE_PRIMARY_URL")
}

connection "redis" "cache" {
  url = env("REDIS_URL")
}
```

**`routes/users.hcl`:**

```hcl
endpoint "GET /api/v1/users/{id}" {
  pipeline {
    sql "find_user" {
      connection = connection.postgres.primary
      query      = <<-SQL
        SELECT id, name, email
        FROM users
        WHERE id = @id
      SQL
      args = {
        id = ctx.request.path.id
      }
    }

    respond {
      condition = steps.find_user.rows_affected == 0
      status    = 404
      body      = { error = "User not found" }
    }

    respond {
      status = 200
      body   = steps.find_user.result
    }
  }
}
```

### 3. Versioned API layout

Organizes endpoint pipelines by API release version:

```text
gateway/
├── Hclapifile
├── schemas/
│   ├── v1.hcl
│   └── v2.hcl
└── routes/
    ├── v1/
    │   ├── auth.hcl
    │   └── billing.hcl
    └── v2/
        ├── auth.hcl
        └── billing.hcl
```

Running `hclapi serve -c ./gateway` merges all versioned routes into the HTTP routing multiplexer simultaneously.
