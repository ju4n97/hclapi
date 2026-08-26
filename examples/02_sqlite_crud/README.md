# 02_sqlite_crud

Demonstrates complete CRUD operations backed by an embedded SQLite database with schema validation and conditional 404 responses.

## Running

```sh
./setup.sh
```

## Testing

### Create a todo

```sh
curl -i -X POST http://localhost:8080/api/v1/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Implement Starlark bindings"}'
```

### List all todos

```sh
curl -i http://localhost:8080/api/v1/todos
```

### Fetch single todo

```sh
curl -i http://localhost:8080/api/v1/todos/1
```

### Update todo

```sh
curl -i -X PUT http://localhost:8080/api/v1/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"completed": true}'
```

### Delete todo

```sh
curl -i -X DELETE http://localhost:8080/api/v1/todos/1
```
