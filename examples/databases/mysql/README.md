# mysql_crud

Demonstrates full CRUD operations and stored procedure execution (`CALL award_member_points`) backed by MySQL 8 with `catch "1062"` duplicate key handling.

## Running

Start the database:

```sh
docker compose up -d
```

Then run the server:

```sh
hclapi serve -c ./examples/databases/mysql
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/databases/mysql
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

# 6. Execute stored procedure (CALL award_member_points)
curl -i -X POST http://localhost:8080/api/v1/members/1/reward \
  -H "Content-Type: application/json" \
  -d '{"bonus": 250}'

# 7. Delete member
curl -i -X DELETE http://localhost:8080/api/v1/members/1
```
