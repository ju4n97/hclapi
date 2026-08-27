# Example 01: Zero dependency

Demonstrates in-memory execution, request manipulation, and response generation using Starlark without external database dependencies.

## Manifest specification

```hcl
server {
  host = "127.0.0.1"
  port = 8080
}

endpoint "GET /api/v1/health" {
  description = "Returns runtime status and system telemetry."

  pipeline {
    starlark "sysinfo" {
      source = <<-STARLARK
        def execute(ctx):
            return {
                "status": "healthy",
                "engine": "hclapi",
                "timestamp": ctx.timestamp_epoch
            }
      STARLARK
    }

    respond {
      status = 200
      body   = steps.sysinfo.result
    }
  }
}
```
