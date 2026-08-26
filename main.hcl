endpoint "GET /ping" {
  description = "A simple ping endpoint"

  pipeline {
    respond {
      status = 200
      body   = "{\"message\": \"pong\"}"
    }
  }
}

endpoint "GET /status" {
  pipeline {
    respond {
      status = 202
      body   = "{\"status\": \"processing\"}"
    }
  }
}