openapi {
  title       = "Modular Production API"
  version     = "1.0.0"
  description = "Demonstrates a multi-file architecture with schemas and routes."
}

endpoint "GET /openapi.json" {
  openapi "spec" {
    format = "json"
  }
}

endpoint "GET /docs" {
  openapi "ui" {
    renderer = "swagger"
  }
}