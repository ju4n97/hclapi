# hclapi

hclapi is a backend engine distributed as a single binary. It turns HashiCorp Configuration Language (HCL) manifests into HTTP APIs, combining data access, business logic, validation, and API definitions in a single declarative configuration, with built-in OpenAPI generation.

[Documentation](https://ju4n97.github.io/hclapi/) ·
[Why hclapi](https://ju4n97.github.io/hclapi/why.html) ·
[Examples](./examples)

## Example

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
}

schema "user_create" {
  field "name"  { type = string, required = true }
  field "email" { type = string, required = true, format = "email" }
}

endpoint "POST /api/v1/users" {
  request {
    body = schema.user_create
  }

  pipeline {
    starlark "normalize" {
      source = <<-STARLARK
        def execute(ctx):
          return {
            "name": ctx.request.body.name.strip(),
            "email": ctx.request.body.email.strip().lower()
          }
      STARLARK
    }

    sql "insert_user" {
      connection = connection.postgres.main
      query      = <<-SQL
        INSERT INTO users (name, email)
        VALUES (@name, @email)
        RETURNING id, name, email, created_at
      SQL
      args = {
        name  = steps.normalize.result.name
        email = steps.normalize.result.email
      }

      catch "23505" {
        abort_with_status = 409
        body = { error = "Email address already registered" }
      }
    }

    respond {
      status = 201
      body   = steps.insert_user.result
    }
  }
}
```

## Install

```sh
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

```sh
hclapi serve -c ./api
```

See [Installation](https://ju4n97.github.io/hclapi/installation.html) for
precompiled binaries, and [Quickstart](https://ju4n97.github.io/hclapi/quickstart.html)
for a full walkthrough.

## Embedding

hclapi implements the standard library `http.Handler` interface and mounts
onto any Go multiplexer.

```go
engine, err := hclapi.NewEngine(hclapi.Options{
    ConfigPath:  "./api",
    StrictTyping: true,
})
if err != nil {
    log.Fatal(err)
}

mux := http.NewServeMux()
mux.Handle("/api/v1/", engine.Handler())
http.ListenAndServe(":8080", mux)
```

See [Go integration](https://ju4n97.github.io/hclapi/go/README.html) for
error handling, logging, and registering native Go steps.

## Documentation

Full reference: request lifecycle, manifest syntax, pipeline steps, and
patterns, at <https://ju4n97.github.io/hclapi/>.

## License

[MIT](LICENSE)
