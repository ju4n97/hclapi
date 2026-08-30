# Registering steps

Registers a Go function that a `go` step can invoke from a manifest.

## Signature

```go
type StepHandler func(ctx *hclapi.Context) (any, error)

err := engine.RegisterStep("namespace.function_name", handler)
```

## Context data access

`*hclapi.Context` is the Go equivalent of [`ctx`](../concepts/context.md).

| Field                 | Type                         | Description                          |
| :-------------------- | :--------------------------- | :----------------------------------- |
| `ctx.Args`            | `map[string]any`             | evaluated `args` from the `go` block |
| `ctx.Request.Method`  | `string`                     | HTTP method                          |
| `ctx.Request.Path`    | `map[string]string`          | route path parameters                |
| `ctx.Request.Query`   | `map[string]string`          | query string parameters              |
| `ctx.Request.Headers` | `map[string]string`          | lowercase headers                    |
| `ctx.Request.Body`    | `any`                        | unmarshaled JSON body                |
| `ctx.Steps`           | `map[string]core.StepResult` | outputs from prior steps             |
| `ctx.TimestampEpoch`  | `int64`                      | ingress timestamp                    |
| `ctx.RawRequest`      | `*http.Request`              | the underlying request               |

## Example: outbound HTTP call

```go
type WeatherResponse struct {
    Current struct {
        Temperature float64 `json:"temperature_2m"`
        Humidity    int     `json:"relative_humidity_2m"`
    } `json:"current"`
}

httpClient := &http.Client{Timeout: 5 * time.Second}

engine.RegisterStep("services.fetch_weather", func(ctx *hclapi.Context) (any, error) {
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

    var data WeatherResponse
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("failed to decode weather API response: %w", err)
    }

    return map[string]any{
        "temperature_c": data.Current.Temperature,
        "humidity_pct":  data.Current.Humidity,
        "fetched_at":    ctx.TimestampEpoch,
    }, nil
})
```

```hcl
endpoint "GET /api/v1/locations/{city}/weather" {
  pipeline {
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

    go "weather" {
      use = "services.fetch_weather"
      args = {
        latitude  = steps.get_coords.result.latitude
        longitude = steps.get_coords.result.longitude
      }
    }

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

## Errors and panics

A returned error stops the pipeline and passes through the normal error
handling path. A panic is recovered by the engine, logged with a stack
trace, and returned as a 500. The server process continues running.
