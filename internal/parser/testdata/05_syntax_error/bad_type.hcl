endpoint "GET /bad" {
  pipeline {
    respond {
      status = "not-an-integer"
    }
  }
}