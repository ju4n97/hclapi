---
title: Quickstart
description: Install Hclapi, create a manifest, and serve an HTTP endpoint in under five minutes.
---

# Quickstart

This guide covers acquiring the Hclapi binary, creating a basic manifest file with multiple endpoints, running the server, and verifying the responses using standard HTTP tooling.

## Installation

Hclapi is distributed as a single standalone executable.

### Precompiled binaries

Download the precompiled binary for the target architecture from the official releases:

```bash
# Example for Linux AMD64
curl -sSL https://github.com/ju4n97/hclapi/releases/latest/download/hclapi_linux_amd64.tar.gz | tar -xz
sudo mv hclapi /usr/local/bin/
```

### Build from source

Alternatively, install directly using the Go toolchain:

```bash
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

Verify that the binary is available in the system path:

```bash
hclapi --help
```

## Manifest definition

Create an empty directory and add a `Hclapifile`:

```bash
mkdir hclapi-quickstart && cd hclapi-quickstart
touch Hclapifile
```

Add the following configuration to `Hclapifile`. This manifest defines two endpoints: a health probe and a payload transformation pipeline executing a sandboxed Starlark script:

```hcl
server {
  host = "127.0.0.1"
  port = 8080
}

endpoint "GET /api/v1/health" {
  description = "Service health check and timestamp telemetry"

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
  description = "Normalizes and prefixes an incoming array of string tags"

  pipeline {
    starlark "sanitize" {
      source = <<-STARLARK
        def execute(ctx):
          prefix = ctx.request.body.get("prefix", "item")
          raw_tags = ctx.request.body.get("tags", [])
          cleaned = [prefix + ":" + t.strip().lower() for t in raw_tags if len(t.strip()) > 0]
          return {
            "count": len(cleaned),
            "tags": cleaned
          }
      STARLARK
    }

    respond {
      status = 200
      body   = steps.sanitize.result
    }
  }
}
```

## Starting the server

Run the server by pointing the `serve` command to the manifest file or containing directory:

```bash
hclapi serve -c .
```

By default, the server boots on the host and port specified in the manifest (`127.0.0.1:8080`).

### Configuration overrides (CLI flags & environment variables)

Network bindings can be overridden without modifying the manifest, adhering to 12-factor application standards.

**Via CLI flags:**

```bash
hclapi serve -c . --port 9000 --host 0.0.0.0
```

**Via environment variables:**

```bash
PORT=9000 HOST=0.0.0.0 hclapi serve -c .
```

The configuration precedence is resolved in the following order:

CLI Flags > Environment Variables > Manifest > Defaults

Detailed command-line options and environment variable bindings are documented in the [CLI reference](/cli/serve).

## Verifying the endpoints

### 1. Health check request

In a separate terminal, test the `GET /api/v1/health` endpoint:

```bash
curl -i http://localhost:8080/api/v1/health
```

Expected HTTP response:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "healthy",
  "timestamp": 1771968000
}
```

### 2. Payload transformation request

Test the `POST /api/v1/transform` endpoint with a JSON body:

```bash
curl -i -X POST http://localhost:8080/api/v1/transform \
  -H "Content-Type: application/json" \
  -d '{"prefix": "env", "tags": [" PROD ", "web", "", " US-EAST "]}'
```

Expected HTTP response:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "count": 3,
  "tags": [
    "env:prod",
    "env:web",
    "env:us-east"
  ]
}
```

### 3. Invalid payload handling (RFC 9457)

Sending malformed JSON to the endpoint triggers an automatic RFC 9457 Problem Details error response before pipeline execution begins:

```bash
curl -i -X POST http://localhost:8080/api/v1/transform \
  -H "Content-Type: application/json" \
  -d '{"tags": ["invalid" "json"]}'
```

Expected HTTP response:

```http
HTTP/1.1 400 Bad Request
Content-Type: application/problem+json

{
  "type": "https://github.com/ju4n97/hclapi/errors/bad-request",
  "title": "Invalid Request Payload",
  "status": 400,
  "detail": "invalid JSON payload: invalid character '\"' after array element",
  "instance": "/api/v1/transform"
}
