# 05_go_embedded

Demonstrates embedding `hclapi` as a library inside an existing Go application, mounting onto `http.ServeMux`, and registering thread-safe native Go step handlers.

## Running

```sh
cd examples/05_go_embedded
go run main.go
```

## Testing

```sh
curl -i http://localhost:8080/api/v1/weather/tokyo

# or

curl -i http://localhost:8080/api/v1/weather/london

# or

curl -i http://localhost:8080/api/v1/weather/new-york

# or

curl -i http://localhost:8080/api/v1/weather/bogota
```
