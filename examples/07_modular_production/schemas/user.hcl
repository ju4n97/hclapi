schema "user_profile" {
  field "id" {
    type = int
  }
  field "name" {
    type     = string
    required = true
  }
  field "email" {
    type     = string
    required = true
    format   = "email"
  }
}