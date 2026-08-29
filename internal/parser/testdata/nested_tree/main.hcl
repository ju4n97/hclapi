endpoint "GET /ping" {
  pipeline {
    respond {
      status = 200
    }
  }
}