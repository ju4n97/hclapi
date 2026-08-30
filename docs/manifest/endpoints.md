---
title: endpoint
description: Bind an HTTP method and path to request validation and a pipeline.
---

# endpoint

Binds an HTTP method and path to a request schema and a pipeline.

## Declaration

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

## Attributes

| Attribute                      | Type     | Required | Description                                                    |
| :----------------------------- | :------- | :------- | :------------------------------------------------------------- |
| route label (`"METHOD /path"`) | `string` | yes      | HTTP method and path pattern                                   |
| `description`                  | `string` | no       | used in logs and generated OpenAPI documentation               |
| `auth`                         | `list`   | no       | route-level authentication guards; `[]` marks the route public |
| `request`                      | `block`  | no       | validation for path, query, headers, and body                  |
| `pipeline`                     | `block`  | yes      | the steps that handle the request                              |

Supported methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`.

## Path templates

A single segment binds to `ctx.request.path.<param>`.

```hcl
endpoint "GET /api/v1/organizations/{org_id}/members/{member_id}" {
  pipeline {
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

A trailing `...` matches all remaining segments.

```hcl
endpoint "GET /static/{filepath...}" {
  pipeline {
    respond {
      status = 200
      body   = { path = ctx.request.path.filepath }
    }
  }
}
```

## Request validation

The `request` block validates path, query, headers, and body independently, using the same `field` syntax as [schema](./schemas.md).

```hcl
endpoint "GET /api/v1/search" {
  request {
    headers {
      field "x-api-key" { type = string, required = true }
    }
    query {
      field "query" { type = string, required = true }
      field "limit" { type = int, default = 20 }
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

| Sub-block | Target                | Validates                      |
| :-------- | :-------------------- | :----------------------------- |
| `path`    | `ctx.request.path`    | route parameter types          |
| `query`   | `ctx.request.query`   | query string, applies defaults |
| `headers` | `ctx.request.headers` | required headers and formats   |
| `body`    | `ctx.request.body`    | JSON body against a schema     |

## Authentication overrides

An endpoint inherits global authentication guards unless it opts out explicitly.

```hcl
endpoint "GET /health/live" {
  description = "Bypassed by load balancers"
  auth        = []

  pipeline {
    respond {
      status = 200
      body   = { status = "healthy", timestamp = ctx.timestamp_epoch }
    }
  }
}
```
