# Example 06: Go embedded plugin

Demonstrates embedding the Hclapi engine within an idiomatic Go service while registering custom native Go functions for outbound HTTP requests.

## Manifest specification

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
}

endpoint "GET /api/v1/weather/{city}" {
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
```

## Host Go code

```go
package main

import (
	"log"
	"net/http"

	"github.com/ju4n97/hclapi"
)

func main() {
	engine, err := hclapi.NewEngine(hclapi.Options{
		ManifestDir:  ".",
		StrictTyping: true,
	})
	if err != nil {
		log.Fatalf("Fatal: engine initialization failed: %v", err)
	}

	engine.RegisterStep("services.get_weather", func(ctx *hclapi.Context) (any, error) {
		city := ctx.Args["city"].(string)
		// Perform outbound HTTP request
		return map[string]any{
			"city":          city,
			"temperature_c": 21.5,
			"condition":     "Clear sky",
		}, nil
	})

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", engine.Handler())

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```
