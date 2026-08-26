# 07_modular_production

Demonstrates a multi-file architecture with primary/replica database connection pooling, global JWT authorization guards, and decoupled routes.

## Running

```sh
docker compose up -d
```

## Testing

```sh
# Generate valid HS256 JWT using secret 'prod_secret_key_123'
JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOjEsImV4cCI6MTk5OTk5OTk5OX0.zXvUdfj0Y7RzZ8_Zg0c3C1mXvUdfj0Y7RzZ8_Zg0c3C"

# Access protected user profile route
curl -i http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer ${JWT_TOKEN}"

# Access protected orders route
curl -i http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer ${JWT_TOKEN}"
```
