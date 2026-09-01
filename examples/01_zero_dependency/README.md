# 01_zero_dependency

Demonstrates in-memory execution, Starlark data transformations, and interactive documentation with zero database dependencies.

## Running

```sh
hclapi serve -c ./examples/01_zero_dependency
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/01_zero_dependency
```

## Testing

```sh
# 1. Open documentation in your browser
open http://localhost:8080/docs

# 2. Health check
curl -i http://localhost:8080/api/v1/health

# 3. Payload transformation
curl -i -X POST http://localhost:8080/api/v1/sanitize \
  -H "Content-Type: application/json" \
  -d '{"prefix": "env", "tags": [" PROD ", "web", "web", " US-EAST "]}'
```
