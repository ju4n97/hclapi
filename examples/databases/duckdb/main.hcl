server {
  host = "127.0.0.1"
  port = 8080

  openapi {
    title       = "DuckDB In-Memory Analytics"
    version     = "1.0.0"
    description = "In-process embedded columnar analytics with zero external servers."
  }
}

connection "duckdb" "main" {
  url = ":memory:"
}

schema "event_ingest" {
  field "user_id" { type = int, required = true }
  field "path" { type = string, required = true }
  field "duration_ms" { type = int, required = true, min = 0 }
  field "country" { type = string, default = "US" }
}

endpoint "GET /docs" {
  openapi {
    ui = "scalar"
  }
}

endpoint "GET /openapi.json" {
  openapi {
    format = "json"
  }
}

endpoint "POST /api/v1/events" {
  description = "Ingests a telemetry event into embedded DuckDB."

  request {
    body = schema.event_ingest
  }

  pipeline {
    sql "init_table" {
      connection = connection.duckdb.main
      query      = <<-SQL
        CREATE TABLE IF NOT EXISTS analytics_events (
          user_id BIGINT,
          path VARCHAR,
          duration_ms INTEGER,
          country VARCHAR,
          timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
      SQL
    }

    sql "ingest" {
      connection = connection.duckdb.main
      query      = <<-SQL
        INSERT INTO analytics_events (user_id, path, duration_ms, country)
        VALUES (@user_id, @path, @duration_ms, @country)
      SQL
      args = {
        user_id     = ctx.request.body.user_id
        path        = ctx.request.body.path
        duration_ms = ctx.request.body.duration_ms
        country     = ctx.request.body.country
      }
    }

    respond {
      status = 201
      body   = { message = "Event recorded in DuckDB", rows_affected = steps.ingest.rows_affected }
    }
  }
}

endpoint "GET /api/v1/analytics/summary" {
  description = "Calculates in-memory columnar metrics across paths."

  pipeline {
    sql "aggregate" {
      connection = connection.duckdb.main
      query      = <<-SQL
        SELECT
          path,
          count(*) AS total_requests,
          count(DISTINCT user_id) AS unique_visitors,
          avg(duration_ms) AS avg_duration_ms
        FROM analytics_events
        GROUP BY path
        ORDER BY total_requests DESC
      SQL
    }

    respond {
      status = 200
      body   = steps.aggregate.rows
    }
  }
}