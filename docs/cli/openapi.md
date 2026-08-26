# OpenAPI generation

Compiling active manifests into an OpenAPI v3 specification.

## Usage

```sh
hclapi openapi [path] > openapi.yaml
```

## Specification mapping

* `endpoint "POST /users/{id}"` $\rightarrow$ OpenAPI Path Item with `post` operation and `id` path parameter.
* `schema` definitions $\rightarrow$ OpenAPI Components (`#/components/schemas/...`).
* `request.body` $\rightarrow$ OpenAPI `requestBody` with JSON schema reference.
* `respond` blocks $\rightarrow$ OpenAPI `responses` keyed by HTTP status code.
