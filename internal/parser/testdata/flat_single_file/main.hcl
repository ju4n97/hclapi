endpoint "GET /health" {
  pipeline {
    respond {
      status = 200
      body   = "OK"
    }
  }
}