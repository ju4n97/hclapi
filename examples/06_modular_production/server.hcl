server {
  host          = "0.0.0.0"
  port          = 8080
  read_timeout  = "15s"
  write_timeout = "30s"
  max_body_size = "10MB"

  openapi {
    title       = "Modular Production API"
    version     = "1.0.0"
    description = "Demonstrates a multi-file architecture with schemas and routes."
  }
}

endpoint "GET /openapi.json" {
  openapi {
    format = "json"
  }
}

endpoint "GET /docs" {
  openapi {
    ui = "swagger"
  }
}

