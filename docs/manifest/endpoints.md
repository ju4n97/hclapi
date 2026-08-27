---
title: Endpoints & route patterns
description: Declaring HTTP endpoints, methods, URI path templates, query/header validation, and pipeline bindings.
---

# Endpoints & route patterns

The `endpoint` block binds an HTTP method and URI path template to a validation contract and an execution pipeline.

## Block declaration

An `endpoint` block requires a single label containing the uppercase **HTTP verb** and the **path pattern**:

```hcl
endpoint "GET /api/v1/accounts/{id}" {
  description = "Fetches account details by identifier"

  request {
    path {
      field "id" {
        type     = int
        required = true
      }
    }
  }

  pipeline {
    sql "find_account" {
      connection = connection.postgres.main
      query      = "SELECT id, name, tier FROM accounts WHERE id = @id"
      args       = { id = ctx.request.path.id }
    }

    respond {
      condition = steps.find_account.rows_affected == 0
      status    = 404
      body      = { error = "Account not found" }
    }

    respond {
      status = 200
      body   = steps.find_account.result
    }
  }
}
```

## Attribute reference

| Attribute / Block | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| **`name`** (Label) | `string` | Yes | HTTP method and URI path declaration (e.g. `"POST /items"`). |
| **`description`** | `string` | No | Operational documentation used for logs and OpenAPI specification generation. |
| **`auth`** | `list` | No | Route-level authentication guard definitions. Set to `[]` to explicitly mark a route public. |
| **`request`** | `block` | No | Schema definitions for path variables, query parameters, headers, and request bodies. |
| **`pipeline`** | `block` | Yes | The ordered execution block containing data steps and response handlers. |

## Supported HTTP methods

* `GET`
* `POST`
* `PUT`
* `PATCH`
* `DELETE`
* `OPTIONS`
* `HEAD`

## URI path templates

Route paths follow standard Go 1.22+ and OpenAPI URI templates:

### Single parameter matching

Matches a single path segment between slashes and binds the value to `ctx.request.path.<param>`:

```hcl
endpoint "GET /api/v1/organizations/{org_id}/members/{member_id}" {
  pipeline {
    # Access via ctx.request.path.org_id and ctx.request.path.member_id
    respond {
      status = 200
      body = {
        org    = ctx.request.path.org_id
        member = ctx.request.path.member_id
      }
    }
  }
}
```

### Catch-all wildcards

A trailing wildcard matches all remaining path segments:

```hcl
endpoint "GET /static/{filepath...}" {
  pipeline {
    # Access full relative path via ctx.request.path.filepath
    respond {
      status = 200
      body   = { path = ctx.request.path.filepath }
    }
  }
}
```

## Request validation sub-blocks (`request`)

The `request` block configures structural type validation across all four HTTP request inputs:

```hcl
endpoint "GET /api/v1/search" {
  request {
    headers {
      field "x-api-key" {
        type     = string
        required = true
      }
    }
    query {
      field "query" {
        type     = string
        required = true
      }
      field "limit" {
        type    = int
        default = 20
      }
    }
  }

  pipeline {
    respond {
      status = 200
      body = {
        search_query = ctx.request.query.query
        page_size    = ctx.request.query.limit
      }
    }
  }
}
```

| Sub-block | Context target | Description |
| :--- | :--- | :--- |
| **`path`** | `ctx.request.path` | Validates and coerces route parameters (e.g. verifying `{id}` is an integer). |
| **`query`** | `ctx.request.query` | Validates query string parameters and applies defaults. |
| **`headers`** | `ctx.request.headers` | Enforces required headers and formats. |
| **`body`** | `ctx.request.body` | Validates JSON payload structure against a schema. |

## Authentication overrides

When global authentication guards are enabled at the server level, endpoints inherit the authentication requirement automatically. An endpoint can explicitly opt out of global authentication by setting `auth = []`:

```hcl
endpoint "GET /health/live" {
  description = "Public health check endpoint bypassed by load balancers"

  # Explicit override: empty list removes global authentication requirements
  auth = []

  pipeline {
    respond {
      status = 200
      body   = { status = "healthy", timestamp = ctx.timestamp_epoch }
    }
  }
}
```
