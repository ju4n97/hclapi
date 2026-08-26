# 03_postgres_transactions

Demonstrates multi-table atomic transactions with automatic rollback and direct error code mapping for PostgreSQL unique constraint violations (`23505`).

## Running

```sh
docker compose up -d
```

## Testing

```sh
# 1. Onboard a new user (Success -> 201)
curl -i -X POST http://localhost:8080/api/v1/onboard \
  -H "Content-Type: application/json" \
  -d '{"email": "jane@example.com", "full_name": "Jane Doe"}'

# 2. Attempt duplicate registration (Conflict -> 409)
curl -i -X POST http://localhost:8080/api/v1/onboard \
  -H "Content-Type: application/json" \
  -d '{"email": "jane@example.com", "full_name": "Jane Doe"}'
```
