server {
  host = "127.0.0.1"
  port = 8080
}

endpoint "GET /openapi.json" {
  openapi {
    format = "json"
  }
}

endpoint "GET /docs" {
  description = "Interactive API reference."

  openapi {
    ui = "scalar"
  }
}

endpoint "GET /api/v1/health" {
  description = "Returns server telemetry and current epoch timestamp."

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
  description = "Demonstrates string trimming and list deduplication in Starlark."

  pipeline {
    starlark "format_tags" {
      source = <<-STARLARK
        def execute(ctx):
            body = ctx.request.body or {}
            prefix = body.get("prefix", "tag")
            raw_tags = body.get("tags", [])

            cleaned = list(set([
                prefix + ":" + t.strip().lower()
                for t in raw_tags
                if len(t.strip()) > 0
            ]))

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