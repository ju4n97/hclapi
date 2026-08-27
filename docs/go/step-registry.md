---
title: Registering custom Go steps
description: Defining type-safe Go functions and exposing them to HCL manifests.
---

# Registering custom Go steps

The `go` pipeline step allows declarative HCL manifests to invoke compiled Go functions. This mechanism bridges declarative pipelines with native Go SDKs, third-party HTTP clients, proprietary business logic, and cryptographic libraries.

## Registration signature

Custom Go step functions match the `StepHandler` signature and are registered on the engine prior to serving traffic:

```go
type StepHandler func(ctx *hclapi.Context) (any, error)
```

The registration method binds a unique string identifier to the handler:

```go
err := engine.RegisterStep("namespace.function_name", handler)
```

## Context data access

The `*hclapi.Context` argument provides structured access to all execution namespaces:

| Method / Field | Type | Description |
| :--- | :--- | :--- |
| `ctx.Args` | `map[string]any` | Dynamically evaluated arguments passed from the HCL `args` block. |
| `ctx.Request.Method` | `string` | HTTP request method. |
| `ctx.Request.Path` | `map[string]string` | Extracted route path parameters. |
| `ctx.Request.Query` | `map[string]string` | Extracted query string parameters. |
| `ctx.Request.Headers` | `map[string]string` | Lowercase HTTP request headers. |
| `ctx.Request.Body` | `any` | Unmarshaled JSON request body. |
| `ctx.Steps` | `map[string]core.StepResult` | Outputs produced by preceding steps in the pipeline. |
| `ctx.TimestampEpoch` | `int64` | Ingress Unix timestamp in seconds. |
| `ctx.RawRequest` | `*http.Request` | The underlying raw Go `http.Request` pointer. |

## Implementation example: Outbound HTTP service

### 1. Go implementation and registration

```go
package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "time"

    "github.com/ju4n97/hclapi"
)

type WeatherResponse struct {
    Current struct {
        Temperature float64 `json:"temperature_2m"`
        Humidity    int     `json:"relative_humidity_2m"`
    } `json:"current"`
}

func main() {
    engine, err := hclapi.NewEngine(hclapi.Options{
        ManifestDir: "./config",
    })
    if err != nil {
        panic(err)
    }

    httpClient := &http.Client{Timeout: 5 * time.Second}

    // Register outbound weather service step
    err = engine.RegisterStep("services.fetch_weather", func(ctx *hclapi.Context) (any, error) {
        lat, okLat := ctx.Args["latitude"].(float64)
        lon, okLon := ctx.Args["longitude"].(float64)
        if !okLat || !okLon {
            return nil, errors.New("missing or invalid 'latitude' or 'longitude' arguments")
        }

        url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,relative_humidity_2m", lat, lon)
        resp, err := httpClient.Get(url)
        if err != nil {
            return nil, fmt.Errorf("external weather API request failed: %w", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return nil, fmt.Errorf("weather API returned HTTP %d", resp.StatusCode)
        }

        var data WeatherResponse
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
            return nil, fmt.Errorf("failed to decode weather API response: %w", err)
        }

        // Returned map is populated into steps.<step_name>.result
        return map[string]any{
            "temperature_c": data.Current.Temperature,
            "humidity_pct":  data.Current.Humidity,
            "fetched_at":    ctx.TimestampEpoch,
        }, nil
    })
    if err != nil {
        panic(err)
    }

    http.ListenAndServe(":8080", engine.Handler())
}
```

### 2. Invocation in HCL manifest

```hcl
endpoint "GET /api/v1/locations/{city}/weather" {
  pipeline {
    # 1. Resolve coordinates from database
    sql "get_coords" {
      connection = connection.postgres.main
      query      = "SELECT latitude, longitude FROM locations WHERE city = @city"
      args       = { city = ctx.request.path.city }
    }

    respond {
      condition = steps.get_coords.rows_affected == 0
      status    = 404
      body      = { error = "Location not found" }
    }

    # 2. Invoke the registered Go step using coordinates from Step 1
    go "weather" {
      use = "services.fetch_weather"
      args = {
        latitude  = steps.get_coords.result.latitude
        longitude = steps.get_coords.result.longitude
      }
    }

    # 3. Return combined result
    respond {
      status = 200
      body = {
        city        = ctx.request.path.city
        temperature = steps.weather.result.temperature_c
        humidity    = steps.weather.result.humidity_pct
      }
    }
  }
}
```

## Error handling and panic recovery

If a registered Go function returns an error, the pipeline execution terminates immediately and triggers the error handling subsystem.

The engine executes all registered Go functions inside a recovery wrapper. If a Go step panics, the panic is safely captured, converted to an error, logged with stack traces, and returned to the client as an RFC 9457 Problem Details 500 error response without crashing the application process.
