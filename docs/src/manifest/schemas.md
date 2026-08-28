# schema

Declares structural validation for incoming requests. Validation runs
before the pipeline.

## Declaration

```hcl
schema "user_create" {
  field "email" {
    type     = string
    required = true
    format   = "email"
  }

  field "full_name" {
    type     = string
    required = true
  }

  field "role" {
    type    = string
    default = "member"
  }

  field "tags" {
    type     = list(string)
    required = false
  }
}
```

The schema is referenced elsewhere as `schema.<name>`.

## Field attributes

| Attribute | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `type` | `type` | required | expected data type |
| `required` | `bool` | `false` | rejects the request with 422 if absent |
| `default` | `any` | `null` | fallback value if the field is absent |
| `format` | `string` | `null` | built-in format constraint |

### Types

| Signature | JSON representation |
| :--- | :--- |
| `string` | `"hello"` |
| `int` | `42` |
| `float` | `3.14159` |
| `bool` | `true`, `false` |
| `any` | any primitive, array, or object |
| `list(<type>)` | `["admin", "member"]` |
| `map(<type>)` | `{"k1": "v1"}` |

### Formats

| Format | Constraint |
| :--- | :--- |
| `"email"` | RFC 5322 email address |
| `"uuid"` | UUID v4 |
| `"uri"` | RFC 3986 absolute URI |
| `"date-time"` | RFC 3339 timestamp |
| `"ipv4"` | dotted-decimal IPv4 address |

## Binding

A named reference binds a reusable schema.

```hcl
endpoint "POST /api/v1/users" {
  request {
    body = schema.user_create
  }

  pipeline {
    sql "insert_user" {
      connection = connection.postgres.main
      query      = "INSERT INTO users (email, name) VALUES (@email, @name) RETURNING id"
      args = {
        email = ctx.request.body.email
        name  = ctx.request.body.full_name
      }
    }

    respond {
      status = 201
      body   = steps.insert_user.result
    }
  }
}
```

Inline fields declare validation for a single endpoint.

```hcl
endpoint "POST /api/v1/webhooks/stripe" {
  request {
    headers {
      field "stripe-signature" {
        type     = string
        required = true
      }
    }
    body {
      field "event_type" { type = string, required = true }
      field "data"       { type = any,    required = true }
    }
  }

  pipeline {
    respond {
      status = 200
      body   = { acknowledged = true }
    }
  }
}
```

## Validation failure

A payload that violates schema rules never reaches the pipeline. The engine
returns a 422 with a field-level breakdown.

```http
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/problem+json

{
  "type": "https://github.com/ju4n97/hclapi/errors/validation-error",
  "title": "Unprocessable Entity",
  "status": 422,
  "detail": "Request payload failed schema validation constraints",
  "instance": "/api/v1/users",
  "invalid_params": [
    { "name": "email", "reason": "must be a valid email address format" },
    { "name": "full_name", "reason": "field is required" }
  ]
}
```
