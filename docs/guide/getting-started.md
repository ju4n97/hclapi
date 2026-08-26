# Getting started

This guide outlines installation, manifest initialization, and standalone daemon execution.

## System requirements

* Go 1.27 or higher (if compiling from source or embedding as a Go library)
* Linux, macOS, or Windows (`amd64` / `arm64`)

## Installation

Install the standalone binary using the Go toolchain:

```sh
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

Verify the installation:

```sh
hclapi --help
```

## Creating a manifest

Create a file named `Hclapifile` in the application root directory:

```hcl
endpoint "GET /ping" {
  description = "Health check verification endpoint"

  pipeline {
    respond {
      status = 200
      body   = "{\"status\":\"healthy\",\"engine\":\"hclapi\"}"
    }
  }
}
```

## Running the server

Start the standalone HTTP daemon targeting the manifest:

```sh
hclapi serve --config ./Hclapifile
```

The engine compiles the AST and binds routes to port `8080` by default:

```text
time=2026-03-30T10:00:00.000Z level=INFO msg="manifests loaded" endpoints_count=1
time=2026-03-30T10:00:00.001Z level=INFO msg="server started" url=http://localhost:8080
```

Test the running endpoint:

```sh
curl -i http://localhost:8080/ping
```

Expected output:

```http
HTTP/1.1 200 OK
Content-Type: application/json
Date: Mon, 30 Mar 2026 10:00:00 GMT
Content-Length: 35

{"status":"healthy","engine":"hclapi"}
```
