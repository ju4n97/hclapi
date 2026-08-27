---
title: Schemas & payload validation
description: Declaring schema blocks, field types, constraints, and request body validation rules in Hclapi manifests.
---

# Schemas & payload validation

The `schema` block defines structural contracts, expected data types, and validation constraints for incoming HTTP payloads. Schemas validate client input at ingress, preventing malformed data from reaching pipeline execution steps.

## Block declaration

A schema is declared with a unique identifier label and contains one or more `field` blocks:

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

Once declared, the schema can be referenced across any endpoint via `schema.<name>`.

## Field attribute reference

| Attribute | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| **`type`** | `type` | Required | Expected data type constraint for the field. |
| **`required`** | `bool` | `false` | When `true`, requests missing this field are rejected with HTTP 422. |
| **`default`** | `any` | `null` | Default fallback value assigned if the field is omitted in the request. |
| **`format`** | `string` | `null` | Built-in string format validator (e.g. `"email"`, `"uuid"`). |

### Supported field types

| Type signature | Description | JSON representation |
| :--- | :--- | :--- |
| `string` | Text string | `"hello"` |
| `int` | Integer number | `42` |
| `float` | Floating-point number | `3.14159` |
| `bool` | Boolean value | `true`, `false` |
| `any` | Generic unconstrained value | Primitive, array, or object |
| `list(<type>)` | Homogeneous array of a single type | `["admin", "member"]` |
| `map(<type>)` | Key-value dictionary of a single type | `{"k1": "v1", "k2": "v2"}` |

### Built-in string formats

| Format | Validation rule | Example |
| :--- | :--- | :--- |
| `"email"` | Valid RFC 5322 email address | `"jane@company.com"` |
| `"uuid"` | Standard UUID v4 format | `"c8f3b2a1-94d2-4b71-bf14-3a7b98d281a1"` |
| `"uri"` | Valid RFC 3986 absolute URI | `"https://api.company.com/v1"` |
| `"date-time"` | RFC 3339 / ISO 8601 timestamp | `"2026-08-26T23:07:00Z"` |
| `"ipv4"` | Valid IPv4 dotted-decimal address | `"192.168.1.1"` |

## Binding schemas to endpoints

Endpoints attach validation rules inside their `request` block.

### 1. Named schema reference

Binds a reusable top-level schema to the endpoint request body:

```hcl
endpoint "POST /api/v1/users" {
  description = "Creates a user using the shared user_create schema"

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

### 2. Inline field declarations

Declares validation constraints directly within the endpoint for single-use schemas:

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
      field "event_type" {
        type     = string
        required = true
      }
      field "data" {
        type     = any
        required = true
      }
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

## Validation failure responses (HTTP 422)

When an incoming payload violates schema rules, the engine short-circuits execution before the pipeline starts, returning an RFC 9457 Problem Details response:

**Client request:**

```bash
curl -i -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"email": "not-an-email"}'
```

**Engine response:**

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
    {
      "name": "email",
      "reason": "must be a valid email address format"
    },
    {
      "name": "full_name",
      "reason": "field is required"
    }
  ]
}
```
