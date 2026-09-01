
server {
  host = "127.0.0.1"
  port = 8080

  openapi {
    title   = "Acme Documentation Demo"
    version = "1.0.0"

    description = <<-MARKDOWN
      ## Overview
      Demonstrates multiple interactive documentation renderers and raw OpenAPI 3.1 specifications.
    MARKDOWN

    servers = [
      {
        url         = "http://localhost:8080"
        description = "Local development server"
      }
    ]

    tags = [
      {
        name        = "system"
        description = "System health and status endpoints"
      }
    ]

    contact {
      name  = "API Support"
      email = "support@example.com"
      url   = "https://example.com/support"
    }

    license {
      name = "MIT"
      url  = "https://opensource.org/licenses/MIT"
    }
  }
}

endpoint "GET /openapi.json" {
  openapi {
    format = "json"
  }
}

endpoint "GET /openapi.yaml" {
  openapi {
    format = "yaml"
  }
}

endpoint "GET /docs" {
  description = "Scalar interactive documentation portal."

  openapi {
    ui = "scalar"
  }
}

endpoint "GET /docs/elements" {
  description = "Stoplight Elements interactive documentation."

  openapi {
    ui = "elements"
  }
}

endpoint "GET /docs/swagger" {
  description = "Swagger UI documentation."

  openapi {
    ui = "swagger"
  }
}

endpoint "GET /docs/redoc" {
  description = "Redoc interactive documentation."

  openapi {
    ui = "redoc"
  }
}

endpoint "GET /api/v1/ping" {
  description = "Simple ping endpoint."

  pipeline {
    respond {
      status = 200
      body   = { status = "pong" }
    }
  }
}