# 06_modular_production

Demonstrates how `hclapi` walks and merges a multi-file directory tree (`server.hcl`, `connections.hcl`, `schemas/`, `routes/`) into a single unified service.

## Running

Initialize the database:

```sh
./setup.sh
```

Then run the server:

```sh
hclapi serve -c ./examples/06_modular_production
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/06_modular_production
```

## Testing

```sh
# View Docs
open http://localhost:8080/docs

# Health
curl -i http://localhost:8080/health/live

# Create user
curl -i -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"email": "jane@example.com", "full_name": "Jane Doe"}'
```
