connection "postgres" "main" {
  url = env("DATABASE_URL")
}

endpoint "GET /api/v1/weather/:city" {
  description = "Fetches live weather via Go HTTP client and logs the query to PostgreSQL."

  request {
    path {
      field "city" {
        type     = string
        required = true
      }
    }
  }

  pipeline {
    go "fetch_weather" {
      use = "services.get_weather"
      args = {
        city = ctx.request.path.city
      }
    }

    sql "log_query" {
      connection = connection.postgres.main
      query      = <<-SQL
        INSERT INTO weather_logs (city, temperature_c, condition)
        VALUES (@city, @temperature, @condition)
        RETURNING id, city, temperature_c, condition, queried_at
      SQL
      args = {
        city        = steps.fetch_weather.result.city
        temperature = steps.fetch_weather.result.temperature_c
        condition   = steps.fetch_weather.result.condition
      }
    }

    respond {
      status = 200
      body   = steps.log_query.result
    }
  }
}