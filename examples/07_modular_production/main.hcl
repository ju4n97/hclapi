server {
  host = "0.0.0.0"
  port = 8080

  read_timeout  = "15s"
  write_timeout = "15s"

  cors {
    allowed_origins = ["https://dashboard.company.com"]
    allowed_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers = ["Authorization", "Content-Type"]
  }
}

api {
  prefix = "/api/v1"
  auth   = [auth.jwt_bearer]
}