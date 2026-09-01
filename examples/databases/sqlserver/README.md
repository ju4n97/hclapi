# sqlserver_crud

Demonstrates a complete CRUD REST API and SQL Server stored procedure execution (`EXEC award_member_points`) backed by Microsoft SQL Server 2022 with `@p1, @p2` parameter rewriting and `catch "2627"` unique violation handling.

## Running

Start the database:

```sh
docker compose up -d
```

Then run the server:

```sh
hclapi serve -c ./examples/databases/sqlserver
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/databases/sqlserver
```

## Testing

```sh
# 1. View interactive documentation
open http://localhost:8080/docs

# 2. List all members
curl -i http://localhost:8080/api/v1/members

# 3. Create a member
curl -i -X POST http://localhost:8080/api/v1/members \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Smith", "email": "alice@example.com", "tier": "premium"}'

# 4. Fetch single member
curl -i http://localhost:8080/api/v1/members/1

# 5. Partial update
curl -i -X PUT http://localhost:8080/api/v1/members/1 \
  -H "Content-Type: application/json" \
  -d '{"tier": "enterprise"}'

# 6. Execute stored procedure (EXEC award_member_points)
curl -i -X POST http://localhost:8080/api/v1/members/1/reward \
  -H "Content-Type: application/json" \
  -d '{"bonus": 500}'

# 7. Delete member
curl -i -X DELETE http://localhost:8080/api/v1/members/1
```
