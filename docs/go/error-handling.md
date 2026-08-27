---
title: Custom error handlers
description: Overriding API error response formats to match corporate conventions.
---

# Custom error handlers

Hclapi implements RFC 9457 Problem Details (`application/problem+json`) by default. When embedding Hclapi into an existing platform with established error schemas, the default serialization can be replaced with a custom `ErrorHandler`.

## The `ErrorHandler` contract

The error handler signature receives the standard HTTP response writer, the active request pointer, and a structured `ProblemDetails` domain object:

```go
type ErrorHandler func(w http.ResponseWriter, r *http.Request, problem hclapi.ProblemDetails)
```

## The `ProblemDetails` domain structure

The `ProblemDetails` struct provides machine-readable metadata regarding the failure:

```go
type ProblemDetails struct {
 Type          string         // RFC 9457 error URI identifier
 Title         string         // Short summary of the error type
 Status        int            // HTTP status code (400, 404, 500, etc.)
 Detail        string         // Specific human-readable failure reason
 Instance      string         // Request URI path that triggered the error
 Step          string         // Pipeline step identifier where failure occurred
 InvalidParams []InvalidParam // Schema validation errors (field name and reason)
 Extensions    map[string]any // Additional contextual metadata
}
```

## Overriding the error handler

Assign the custom handler function to `ErrorHandler` within `hclapi.Options`:

```go
package main

import (
 "encoding/json"
 "log"
 "net/http"
 "time"

 "github.com/ju4n97/hclapi"
)

func main() {
 engine, err := hclapi.NewEngine(hclapi.Options{
  ManifestDir: "./config",
  
  // Custom error handler mapping RFC 9457 into an internal envelope
  ErrorHandler: func(w http.ResponseWriter, r *http.Request, problem hclapi.ProblemDetails) {
   w.Header().Set("Content-Type", "application/json")
   w.WriteHeader(problem.Status)

   var validationErrors []string
   for _, p := range problem.InvalidParams {
    validationErrors = append(validationErrors, p.Name+": "+p.Reason)
   }

   response := map[string]any{
    "success": false,
    "error": map[string]any{
     "code":       problem.Status,
     "message":    problem.Detail,
     "type":       problem.Title,
     "path":       problem.Instance,
     "step":       problem.Step,
     "validation": validationErrors,
     "timestamp":  time.Now().UTC().Format(time.RFC3339),
    },
   }

   _ = json.NewEncoder(w).Encode(response)
  },
 })
 if err != nil {
  log.Fatalf("failed to initialize engine: %v", err)
 }

 http.ListenAndServe(":8080", engine.Handler())
}
```

## Error categories handled

The configured `ErrorHandler` processes all failure domains across the engine:

1. **Ingress transport errors (400 Bad Request)**: Malformed JSON syntax, invalid content encodings, or unreadable request streams.
2. **Schema validation errors (422 Unprocessable Entity)**: Missing required fields, invalid regex formats, or type coercion failures.
3. **Pipeline execution failures (500 Internal Server Error)**: Unhandled Starlark runtime exceptions, database driver timeouts, or Go step panics.
