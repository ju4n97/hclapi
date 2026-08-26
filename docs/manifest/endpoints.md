# Endpoints & routing

The `endpoint` block declares an HTTP method, path pattern, input validation rules, and the execution pipeline.

## Syntax

```hcl
endpoint "POST /api/v1/workspaces/{workspace_id}/members" {
  description = "Enrolls a new member into a workspace."

  request {
    path {
      field "workspace_id" {
        type     = int
        required = true
      }
    }
    body {
      schema = schema.member_enrollment
    }
  }

  pipeline {
    # Execution steps
  }
}
```

## Path parameters

Path parameters follow RFC 6570 URI Template conventions using `{name}`. Parameter values are extracted and populated into `ctx.request.path.<name>`.

## Inline request bodies

Request bodies can reference a global schema or declare inline validation rules:

```hcl
request {
  body {
    field "token" {
      type     = string
      required = true
    }
  }
}
```

## Route-level authentication overrides

Endpoints inherit global authentication guards declared in the `api` block unless explicitly overridden:

```hcl
endpoint "POST /api/v1/public/webhook" {
  auth = [] # Explicitly public route override

  pipeline {
    # ...
  }
}
```
