# Custom error handlers

Overrides the default RFC 9457 error format when embedding Hclapi into a
platform with an existing error schema.

## Signature

```go
type ErrorHandler func(w http.ResponseWriter, r *http.Request, problem hclapi.ProblemDetails)

type ProblemDetails struct {
    Type          string
    Title         string
    Status        int
    Detail        string
    Instance      string
    Step          string
    InvalidParams []InvalidParam
    Extensions    map[string]any
}
```

## Example

```go
engine, err := hclapi.NewEngine(hclapi.Options{
    ManifestDir: "./config",

    ErrorHandler: func(w http.ResponseWriter, r *http.Request, problem hclapi.ProblemDetails) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(problem.Status)

        var validationErrors []string
        for _, p := range problem.InvalidParams {
            validationErrors = append(validationErrors, p.Name+": "+p.Reason)
        }

        json.NewEncoder(w).Encode(map[string]any{
            "success": false,
            "error": map[string]any{
                "code":       problem.Status,
                "message":    problem.Detail,
                "type":       problem.Title,
                "path":       problem.Instance,
                "step":       problem.Step,
                "validation": validationErrors,
            },
        })
    },
})
```

## Scope

The configured handler processes every failure domain: 400s from malformed
request bodies, 422s from schema validation, and 500s from unhandled
pipeline exceptions, including Starlark errors, database timeouts, and Go
panics.
