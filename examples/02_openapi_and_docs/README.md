# 02_openapi_and_docs

Demonstrates multiple interactive documentation renderers (Scalar, Elements, Swagger UI, Redoc) and raw OpenAPI 3.1 JSON/YAML endpoints.

## Running

```sh
hclapi serve -c ./examples/02_openapi_and_docs
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/02_openapi_and_docs
```

## Testing

```sh
# View Scalar UI
open http://localhost:8080/docs

# View Stoplight Elements UI
open http://localhost:8080/docs/elements

# View Swagger UI
open http://localhost:8080/docs/swagger

# View Redoc UI
open http://localhost:8080/docs/redoc

# View raw JSON spec
curl http://localhost:8080/openapi.json

# View raw YAML spec
curl http://localhost:8080/openapi.yaml
```
