# Schemas & validation

The `schema` block defines reusable structural contracts for payload validation and OpenAPI specification compilation.

## Syntax

```hcl
schema "user_create" {
  field "username" {
    type       = string
    required   = true
    min_length = 3
    max_length = 32
    pattern    = "^[a-z0-9_]+$"
  }

  field "email" {
    type     = string
    required = true
    format   = "email"
  }

  field "age" {
    type     = int
    min      = 18
    max      = 120
    default  = 18
  }

  field "role" {
    type    = string
    enum    = ["admin", "editor", "viewer"]
    default = "viewer"
  }
}
```

## Supported field types

* `string`: UTF-8 character string.
* `int`: 64-bit integer.
* `float`: 64-bit floating-point number.
* `bool`: Boolean value (`true` or `false`).
* `list(<type>)`: Homogeneous list of typed values.
* `object`: Nested key-value dictionary.
* `any`: Unvalidated JSON structure.

## Validation attributes

| Attribute | Type | Supported Types | Description |
| :--- | :--- | :--- | :--- |
| `required` | `bool` | All | If `true`, field must not be null or omitted. |
| `default` | Any | All | Default value injected if field is omitted. |
| `min` | `int` / `float` | Numbers | Minimum numerical value (inclusive). |
| `max` | `int` / `float` | Numbers | Maximum numerical value (inclusive). |
| `min_length` | `int` | `string`, `list` | Minimum character or element count. |
| `max_length` | `int` | `string`, `list` | Maximum character or element count. |
| `pattern` | `string` | `string` | Regular expression constraint. |
| `enum` | `list(string)` | `string` | Explicit set of permitted values. |
