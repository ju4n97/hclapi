---
title: Go
description: Calling custom registered Go functions inside your declarative pipeline.
---

# Go

The `go` step executes custom native Go functions registered in the host Go application. This step enables integration with proprietary business logic, third-party Go SDKs, external HTTP clients, or hardware cryptography while maintaining declarative pipeline orchestration.

## Block declaration

```hcl
go "<name>" {
  use  = "<registered_function_name>"
  args = {
    city = ctx.request.path.city
  }
}
```

## Attribute reference

| Attribute          | Type     | Required | Description                                                   |
| :----------------- | :------- | :------- | :------------------------------------------------------------ |
| **`name`** (Label) | `string` | Yes      | Unique step identifier used under `steps.<name>.result`.      |
| **`use`**          | `string` | Yes      | The identifier of the Go function registered on the `Engine`. |
| **`args`**         | `map`    | No       | Dynamic arguments evaluated and passed into `ctx.Args`.       |

## Registering Go functions in the host application

Go step handlers match the `func(*hclapi.Context) (any, error)` signature and are registered on the `hclapi.Engine` before starting the server:

```go
package main

import (
 "errors"
 "fmt"
 "net/http"
 "strings"

 "github.com/ju4n97/hclapi"
)

func main() {
  engine, err := hclapi.NewEngine(hclapi.Options{
    ManifestDir: "./config",
  })
  if err != nil {
    panic(err)
  }

  // Register native Go step handler
  err = engine.RegisterStep("services.weather_lookup", func(ctx *hclapi.Context) (any, error) {
    city, ok := ctx.Args["city"].(string)
    if !ok || len(city) == 0 {
      return nil, errors.New("missing or invalid 'city' argument")
    }

    // Perform custom logic, outbound HTTP calls, or proprietary compute
    normalizedCity := strings.ToLower(strings.TrimSpace(city))
    temperatureC := 22.5

    // Return value is stored in steps.<name>.result
    return map[string]any{
      "city":          normalizedCity,
      "temperature_c": temperatureC,
      "condition":     "Sunny",
    }, nil
  })
  if err != nil {
    panic(err)
  }

  http.ListenAndServe(":8080", engine.Handler())
}
```

## Invoking the registered step in HCL

```hcl
endpoint "GET /api/v1/weather/{city}" {
  description = "Fetches live weather via Go client and logs the query to PostgreSQL"

  pipeline {
    # 1. Execute the registered native Go function
    go "fetch_weather" {
      use = "services.weather_lookup"
      args = {
        city = ctx.request.path.city
      }
    }

    # 2. Persist the Go result into PostgreSQL
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

    # 3. Return aggregated result
    respond {
      status = 200
      body   = steps.log_query.result
    }
  }
}
```

## Panic safety and error propagation

The engine wraps all native Go step executions in a deferred panic recovery barrier. If a custom Go function panics (e.g. nil pointer dereference), the engine recovers safely, translates the panic into an error, and terminates the pipeline with an RFC 9457 Problem Details 500 error response without crashing the HTTP server.
