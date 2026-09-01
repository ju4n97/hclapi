schema "user_create" {
  field "email" {
    type     = string
    required = true
    format   = "email"
  }

  field "full_name" {
    type       = string
    required   = true
    min_length = 2
  }
}