# 01_zero_dependency

Demonstrates in-memory execution, request manipulation, and response generation using Starlark without external database dependencies.

## Running

```sh
docker compose up -d
```

## Testing

```sh
# System health check
curl -i http://localhost:8080/api/v1/health

# Payload transformation
curl -i -X POST http://localhost:8080/api/v1/sanitize \
  -H "Content-Type: application/json" \
  -d '{"prefix": "env", "tags": [" PROD ", "web", " US-EAST "]}'
```
