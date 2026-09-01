# sqlite_crud

Demonstrates a complete CRUD REST API and atomic reward mutations backed by an embedded SQLite database using `.rows`, `.row`, relative path resolution, and `catch "19"`.

## Running

Initialize the database:

```sh
./setup.sh
```

Then run the server:

```sh
hclapi serve -c ./examples/databases/sqlite
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/databases/sqlite
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

# 6. Award bonus points
curl -i -X POST http://localhost:8080/api/v1/members/1/reward \
  -H "Content-Type: application/json" \
  -d '{"bonus": 250}'

# 7. Delete member
curl -i -X DELETE http://localhost:8080/api/v1/members/1
```
