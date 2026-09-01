server {
  host = "127.0.0.1"
  port = 8080

  openapi {
    title       = "ClickHouse Analytics API"
    version     = "1.0.0"
    description = "High-throughput telemetry ingestion and columnar aggregations in ClickHouse."
  }
}

connection "clickhouse" "main" {
  url = "clickhouse://default:@127.0.0.1:9000/default"

  pool {
    max_open_conns = 15
  }
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
  description = "Ingests a high-throughput telemetry event."

  request {
    body = schema.event_ingest
  }

  pipeline {
    sql "ingest" {
      connection = connection.clickhouse.main
      query      = <<-SQL
        INSERT INTO analytics_events (event_id, user_id, path, duration_ms, country, timestamp)
        VALUES (generateUUIDv4(), @user_id, @path, @duration_ms, @country, now())
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
      body   = { message = "Event ingested", rows_affected = steps.ingest.rows_affected }
    }
  }
}

endpoint "GET /api/v1/analytics/summary" {
  description = "Calculates total requests, unique visitors, and average latency by path."

  pipeline {
    sql "aggregate" {
      connection = connection.clickhouse.main
      query      = <<-SQL
        SELECT
          path,
          count() AS total_requests,
          uniq(user_id) AS unique_visitors,
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

endpoint "GET /api/v1/analytics/countries" {
  description = "Aggregates traffic volume by geographic region."

  pipeline {
    sql "countries" {
      connection = connection.clickhouse.main
      query      = <<-SQL
        SELECT
          country,
          count() AS event_count
        FROM analytics_events
        GROUP BY country
        ORDER BY event_count DESC
        LIMIT 10
      SQL
    }

    respond {
      status = 200
      body   = steps.countries.rows
    }
  }
}