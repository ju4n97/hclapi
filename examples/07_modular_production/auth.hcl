auth "jwt_bearer" {
  type   = "jwt"
  secret = env("JWT_SECRET")
  header = "Authorization"
  prefix = "Bearer "
}