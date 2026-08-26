endpoint "GET /ping" {
  pipeline {
    respond {
      status = "two hundred" # Syntax error: expects int
    }
  }
}