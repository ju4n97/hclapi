# 03_schemas_and_validation

Demonstrates schema validation, field formats (`email`, `uuid`), bounds, enums, default value injection, and automatic HTTP 422 Problem Details responses.

## Running

```sh
hclapi serve -c ./examples/03_schemas_and_validation
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/03_schemas_and_validation
```

## Testing

### Valid request (201 Created)

```sh
curl -i -X POST "http://localhost:8080/api/v1/users?source=referral" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: f47ac10b-58cc-4372-a567-0e02b2c3d479" \
  -d '{
    "email": "jane@example.com",
    "username": "jane_doe",
    "account_type": "individual",
    "age": 28,
    "tags": ["golang", "api"]
  }'
```

### Invalid payload (422 Unprocessable Entity)

```sh
curl -i -X POST "http://localhost:8080/api/v1/users" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "bad-email",
    "username": "A!",
    "account_type": "invalid_type",
    "age": 14,
    "tags": ["duplicate", "duplicate"]
  }'
```
