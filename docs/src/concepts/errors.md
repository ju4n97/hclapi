# Errors

hclapi returns [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) Problem
Details for every error, whether raised at ingress, during validation, or
during pipeline execution.

```json
{
  "type": "https://github.com/ju4n97/hclapi/errors/invalid-json",
  "title": "Invalid Request Payload",
  "status": 400,
  "detail": "invalid JSON payload: invalid character '\"' after array element",
  "instance": "/api/v1/transform",
  "step": "ingress",
  "invalid_params": [{ "name": "tags", "reason": "must be a valid JSON array" }]
}
```

| Field            | Type     | Description                                             |
| :--------------- | :------- | :------------------------------------------------------ |
| `type`           | `string` | URI identifying the problem category                    |
| `title`          | `string` | short summary of the problem type                       |
| `status`         | `int`    | HTTP status code                                        |
| `detail`         | `string` | explanation specific to this occurrence                 |
| `instance`       | `string` | request path that produced the error                    |
| `step`           | `string` | pipeline step where the failure occurred, if applicable |
| `invalid_params` | `list`   | field-level validation errors                           |

## Ingress errors

Detected before the pipeline begins execution.

```http
HTTP/1.1 400 Bad Request
Content-Type: application/problem+json

{
  "type": "https://github.com/ju4n97/hclapi/errors/bad-request",
  "title": "Invalid Request Payload",
  "status": 400,
  "detail": "invalid JSON payload: invalid character '\"' after array element",
  "instance": "/api/v1/sanitize"
}
```

## Declarative error handling

Business errors, authorization guards, and not-found states are declared
directly in the manifest as conditional `respond` blocks. There is no
separate error-declaration mechanism.

```hcl
respond {
  condition = ctx.request.body.name == null || ctx.request.body.name == ""
  status    = 422
  body = {
    type     = "https://github.com/ju4n97/hclapi/errors/validation-error"
    title    = "Unprocessable Entity"
    status   = 422
    detail   = "The 'name' field is required and cannot be empty."
    instance = ctx.request.path
  }
}
```

When `condition` evaluates to `true`, the response is written and the
pipeline terminates.

## Unhandled runtime failures

An unhandled exception, a Starlark error, a database timeout, is caught,
logged internally with a full trace, and returned as a 500.

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/problem+json

{
  "type": "https://github.com/ju4n97/hclapi/errors/pipeline-execution-failed",
  "title": "Pipeline Execution Error",
  "status": 500,
  "detail": "step \"compute_metrics\" starlark execution failed: runtime error: division by zero",
  "instance": "/api/v1/metrics"
}
```

## Custom formats

When hclapi is embedded in a platform with an existing error schema, the
default format can be replaced with a custom `ErrorHandler`. See
[Custom error handlers](../go/error-handlers.md).
