---
title: Errors
description: RFC 9457 Problem Details returned by hclapi for ingress, validation, and pipeline failures.
---

# Errors

hclapi returns [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) Problem Details for every error, whether raised at ingress, during validation, or during pipeline execution.

```json
{
  "type": "urn:hclapi:error:bad-request",
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
| `type`           | `string` | URI or URN identifying the problem category             |
| `title`          | `string` | Short summary of the problem type                       |
| `status`         | `int`    | HTTP status code                                        |
| `detail`         | `string` | Explanation specific to this occurrence                 |
| `instance`       | `string` | Request path that produced the error                    |
| `step`           | `string` | Pipeline step where the failure occurred, if applicable |
| `invalid_params` | `list`   | Field-level validation errors                           |

## Problem type identifiers

By default, hclapi emits domain-agnostic URN identifiers:

- `urn:hclapi:error:bad-request`: Malformed JSON or unparseable payload.
- `urn:hclapi:error:payload-too-large`: Request body exceeded `max_body_size`.
- `urn:hclapi:error:validation-error`: Payload failed schema constraints.
- `urn:hclapi:error:pipeline-execution-failed`: Unhandled step failure or database timeout.

### Self-hosting error documentation

Standalone binary deployments can override the default URN prefix by setting `error_base_url` in the `server {}` block:

```hcl
server {
  error_base_url = "https://docs.mycompany.com/errors/"
}
```

This transforms `urn:hclapi:error:bad-request` into `https://docs.mycompany.com/errors/bad-request`.

## Ingress errors

Detected before the pipeline begins execution.

```http
HTTP/1.1 400 Bad Request
Content-Type: application/problem+json

{
  "type": "urn:hclapi:error:bad-request",
  "title": "Invalid Request Payload",
  "status": 400,
  "detail": "invalid JSON payload: invalid character '\"' after array element",
  "instance": "/api/v1/sanitize"
}
```

## Payload size errors

Triggered when request payload exceeds `max_body_size`.

```http
HTTP/1.1 413 Payload Too Large
Content-Type: application/problem+json

{
  "type": "urn:hclapi:error:payload-too-large",
  "title": "Request Entity Too Large",
  "status": 413,
  "detail": "request body exceeded maximum size limit of 10MB",
  "instance": "/api/v1/upload"
}
```

## Custom error formatting

When embedding hclapi in Go, the default format can be completely replaced by providing an `ErrorHandler` in `Options`. See [Custom error handlers](../guides/go.md#custom-error-handlers).
