# 06_go_embedded

Demonstrates how to embed hclapi and use the `go` step to perform outbound network requests to external third-party APIs.

## Scenario

1. The client requests current weather for a city: `GET /api/v1/weather/:city`.
2. A native Go function (`services.get_weather`) fetches current conditions from an external weather service.
3. The pipeline records the query and temperature in PostgreSQL.
4. The aggregated data is returned to the client.

## Running

1. Start the PostgreSQL container:

    ```sh
    docker compose up -d
    ```

2. Run the Go application:

    ```sh
    export DATABASE_URL="postgres://hclapi:hclapi_password@localhost:5432/hclapi_db?sslmode=disable"
    go run main.go
    ```

## Testing

```sh
# Fetch and log weather for Tokyo
curl -i http://localhost:8080/api/v1/weather/tokyo

# Fetch and log weather for Bogotá
curl -i http://localhost:8080/api/v1/weather/bogota
```
