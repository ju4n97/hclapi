# Options and logging

`hclapi.Options` configures the engine at initialization.

```go
type Options struct {
    ConfigPath   string
    StrictTyping bool
    ErrorHandler ErrorHandler
    Logger       *slog.Logger
}
```

| Field          | Type           | Default          | Description                                                                     |
| :------------- | :------------- | :--------------- | :------------------------------------------------------------------------------ |
| `ConfigPath`   | `string`       | `"."`            | file or directory containing manifests                                          |
| `StrictTyping` | `bool`         | `false`          | enforces strict schema and expression type checking                             |
| `ErrorHandler` | `ErrorHandler` | RFC 9457 handler | overrides error serialization; see [Custom error handlers](./error-handlers.md) |
| `Logger`       | `*slog.Logger` | discard          | standard `log/slog` logger                                                      |

## Structured logging

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

engine, err := hclapi.NewEngine(hclapi.Options{
    ConfigPath:   "./manifests",
    StrictTyping: true,
    Logger:       logger,
})
```

Emitted events at each lifecycle phase:

```json
{"level":"INFO","msg":"manifests loaded","endpoints_count":12}
{"level":"DEBUG","msg":"registering route","pattern":"GET /api/v1/users/{id}"}
{"level":"WARN","msg":"invalid request payload","error":"invalid JSON payload: syntax error","path":"/api/v1/sanitize"}
{"level":"ERROR","msg":"pipeline execution failed","error":"step \"query_db\" execution failed: connection timeout","path":"/api/v1/users/42"}
```

## Resolved server configuration

```go
srvConfig := engine.Server()

slog.Info("server configuration loaded",
    "host", srvConfig.Host,
    "port", srvConfig.Port,
    "read_timeout", srvConfig.ReadTimeout.String(),
)
```

`srvConfig` reflects the same values [`server`](../manifest/server.md)
resolves after CLI flags and environment variables are applied.
