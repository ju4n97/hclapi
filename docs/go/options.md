---
title: Options & logging
description: Configuring structured slog loggers, strict typing, and manifest paths.
---

# Options & logging

The `hclapi.Options` struct configures compilation parameters, validation strictness, structured telemetry loggers, and error handlers during engine initialization.

## Options reference

```go
type Options struct {
 // ManifestDir specifies the filesystem path containing .hcl or Hclapifile manifests.
 ManifestDir string

 // StrictTyping enforces strict schema validation across all endpoints.
 StrictTyping bool

 // Logger receives structured operational telemetry. If nil, logs are discarded.
 Logger *slog.Logger

 // ErrorHandler overrides API error response serialization. If nil, defaults to RFC 9457.
 ErrorHandler ErrorHandler
}
```

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| **`ManifestDir`** | `string` | `"."` | Path to a single manifest file (`Hclapifile`, `.hcl`) or a root directory containing manifests. |
| **`StrictTyping`** | `bool` | `false` | When enabled, enforces strict type checking on schema definitions and dynamic expressions. |
| **`Logger`** | `*slog.Logger` | Discard | Structured logger instance implementing Go's standard `log/slog`. |
| **`ErrorHandler`** | `ErrorHandler` | RFC 9457 Handler | Callback function executed when an ingress or runtime failure occurs. |

## Structured logging integration (`slog`)

Hclapi integrates natively with Go's standard `log/slog` package, emitting structured log attributes without third-party logging dependencies:

```go
package main

import (
 "log/slog"
 "net/http"
 "os"

 "github.com/ju4n97/hclapi"
)

func main() {
 // 1. Configure structured JSON logger at debug level
 logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
  Level: slog.LevelDebug,
 }))

 // 2. Pass logger into engine options
 engine, err := hclapi.NewEngine(hclapi.Options{
  ManifestDir:  "./manifests",
  StrictTyping: true,
  Logger:       logger,
 })
 if err != nil {
  logger.Error("engine boot failed", "error", err)
  os.Exit(1)
 }

 http.ListenAndServe(":8080", engine.Handler())
}
```

### Emitted log events

Hclapi outputs structured key-value telemetry at distinct lifecycle phases:

* **Initialization (`INFO` / `DEBUG`)**:

  ```json
  {"time":"2026-08-27T00:00:00Z","level":"INFO","msg":"manifests loaded","endpoints_count":12}
  {"time":"2026-08-27T00:00:00Z","level":"DEBUG","msg":"registering route","pattern":"GET /api/v1/users/{id}"}
  ```

* **Ingress validation warnings (`WARN`)**:

  ```json
  {"time":"2026-08-27T00:00:01Z","level":"WARN","msg":"invalid request payload","error":"invalid JSON payload: syntax error","path":"/api/v1/sanitize"}
  ```

* **Pipeline execution errors (`ERROR`)**:

  ```json
  {"time":"2026-08-27T00:00:02Z","level":"ERROR","msg":"pipeline execution failed","error":"step \"query_db\" execution failed: connection timeout","path":"/api/v1/users/42"}
  ```

## Accessing parsed server configuration

The engine parses server settings from the `server { ... }` block in the manifest, applies production defaults, and exposes the resolved configuration via `engine.Server()`:

```go
engine, err := hclapi.NewEngine(hclapi.Options{ManifestDir: "./config"})
if err != nil {
 panic(err)
}

// Retrieve resolved server configuration
srvConfig := engine.Server()

slog.Info("server configuration loaded",
 "host", srvConfig.Host,
 "port", srvConfig.Port,
 "read_timeout", srvConfig.ReadTimeout.String(),
 "write_timeout", srvConfig.WriteTimeout.String(),
 "max_body_size_bytes", srvConfig.MaxBodySize.Bytes(),
)
```
