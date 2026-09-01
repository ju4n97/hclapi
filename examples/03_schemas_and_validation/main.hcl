
server {
  host = "127.0.0.1"
  port = 8080
}

schema "user_create" {
  field "email" {
    type        = string
    required    = true
    format      = "email"
    description = "User primary email address"
  }

  field "username" {
    type       = string
    required   = true
    min_length = 3
    max_length = 20
    pattern    = "^[a-z0-9_]+$"
  }

  field "account_type" {
    type     = string
    required = true
    enum     = ["individual", "business"]
  }

  field "age" {
    type     = int
    required = false
    min      = 18
    max      = 120
  }

  field "role" {
    type    = string
    default = "member"
    enum    = ["admin", "member", "viewer"]
  }

  field "tags" {
    type         = list(string)
    required     = false
    min_items    = 1
    max_items    = 5
    unique_items = true
  }
}

endpoint "POST /api/v1/users" {
  description = "Registers a new user with strict schema validation."

  request {
    headers {
      field "x-api-key" {
        type     = string
        required = true
        format   = "uuid"
      }
    }

    query {
      field "source" {
        type    = string
        default = "direct"
        enum    = ["direct", "referral", "ad"]
      }
    }

    body = schema.user_create
  }

  pipeline {
    respond {
      status = 201
      body = {
        message    = "User validated and created"
        user       = ctx.request.body
        api_key    = ctx.request.headers.x-api-key
        src        = ctx.request.query.source
        created_at = now()
      }
    }
  }
}