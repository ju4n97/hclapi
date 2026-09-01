endpoint "GET /health/live" {
  description = "Public health check endpoint."

  pipeline {
    respond {
      status = 200

      body = {
        status    = "ok"
        timestamp = now()
      }
    }
  }
}