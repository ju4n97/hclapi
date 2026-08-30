# Quickstart

This section creates a manifest, starts the server, and calls two endpoints.

## Create a manifest

```sh
mkdir hclapi-quickstart && cd hclapi-quickstart
touch main.hcl
```

```hcl
server {
  host = "127.0.0.1"
  port = 8080
}

endpoint "GET /api/v1/health" {
  pipeline {
    respond {
      status = 200
      body = {
        status    = "healthy"
        timestamp = ctx.timestamp_epoch
      }
    }
  }
}

endpoint "POST /api/v1/transform" {
  pipeline {
    starlark "sanitize" {
      source = <<-STARLARK
        def execute(ctx):
          prefix = ctx.request.body.get("prefix", "item")
          tags = ctx.request.body.get("tags", [])
          cleaned = [prefix + ":" + t.strip().lower() for t in tags if len(t.strip()) > 0]
          return {"count": len(cleaned), "tags": cleaned}
      STARLARK
    }

    respond {
      status = 200
      body   = steps.sanitize.result
    }
  }
}
```

`GET /api/v1/health` responds unconditionally. `POST /api/v1/transform` runs
a [`starlark`](./steps/starlark.md) step and returns its result. See
[Pipelines and steps](./concepts/pipelines.md) for how steps pass data to
each other.

## Start the server

```sh
hclapi serve -c .
```

The server binds to `127.0.0.1:8080`, as declared in the `server` block.
Host and port can be overridden without editing the manifest; see
[CLI reference](./cli.md).

```sh
hclapi serve -c . --port 9000 --host 0.0.0.0
```

## Call the endpoints

```sh
curl -i http://localhost:8080/api/v1/health
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status": "healthy", "timestamp": 1771968000}
```

```sh
curl -i -X POST http://localhost:8080/api/v1/transform \
  -H "Content-Type: application/json" \
  -d '{"prefix": "env", "tags": [" PROD ", "web", "", " US-EAST "]}'
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"count": 3, "tags": ["env:prod", "env:web", "env:us-east"]}
```

Malformed JSON is rejected before the pipeline runs.

```sh
curl -i -X POST http://localhost:8080/api/v1/transform \
  -d '{"tags": ["invalid" "json"]}'
```

```http
HTTP/1.1 400 Bad Request
Content-Type: application/problem+json

{
  "type": "urn:hclapi:error:bad-request",
  "title": "Invalid Request Payload",
  "status": 400,
  "detail": "invalid JSON payload: invalid character '\"' after array element",
  "instance": "/api/v1/transform"
}
```

This is hclapi's standard error format. See [Errors](./concepts/errors.md).
