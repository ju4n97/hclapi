endpoint "GET /v1/users" {
  pipeline {
    respond {
      status = 200
    }
  }
}

endpoint "POST /v1/users" {
  pipeline {
    respond {
      status = 201
    }
  }
}