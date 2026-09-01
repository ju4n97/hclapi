# 04_sqlite_crud

Demonstrates a complete, production-ready CRUD REST API backed by an embedded SQLite database using `.rows`, `.row`, and `problem()`.

## Running

Initialize the database:

```sh
./setup.sh
```

Then run the server:

```sh
hclapi serve -c ./examples/04_sqlite_crud
```

or

```sh
go run ./cmd/hclapi serve -c ./examples/04_sqlite_crud
```

## Testing

```sh
# 1. List todos
curl -i http://localhost:8080/api/v1/todos

# 2. Create a todo
curl -i -X POST http://localhost:8080/api/v1/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Implement Starlark step"}'

# 3. Fetch single todo
curl -i http://localhost:8080/api/v1/todos/1

# 4. Partial update
curl -i -X PUT http://localhost:8080/api/v1/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title": "Updated Title"}'

# 5. Delete todo
curl -i -X DELETE http://localhost:8080/api/v1/todos/1
```
