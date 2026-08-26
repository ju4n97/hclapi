server {
  host = "0.0.0.0"
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

endpoint "POST /api/v1/sanitize" {
  description = "Demonstrates input parsing and dictionary normalization."

  request {
    body {
      field "tags" {
        type     = list(string)
        required = true
      }
      field "prefix" {
        type    = string
        default = "tag"
      }
    }
  }

  pipeline {
    starlark "format_tags" {
      source = <<-STARLARK
        def execute(ctx):
          prefix = ctx.request.body.get("prefix", "tag")
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
      body   = steps.format_tags.result
    }
  }
}