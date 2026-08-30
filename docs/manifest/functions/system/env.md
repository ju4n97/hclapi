---
title: env
description: Read the value of an environment variable from the host operating system.
---

# env

Reads the value of an environment variable from the host operating system.

## Signature

```hcl
env(name: string) -> string
```

## Parameters

| Parameter | Type     | Required | Description                              |
| :-------- | :------- | :------- | :--------------------------------------- |
| `name`    | `string` | yes      | The environment variable name to look up |

## Return value

Returns a `string` containing the value of the environment variable. If the variable is not set, it returns an empty string `""`.

## Examples

### Database connection configuration

```hcl
connection "postgres" "main" {
  url = env("DATABASE_URL")
}
```

### Dynamic fallback with [coalesce](../collections/coalesce.md)

```hcl
endpoint "GET /health" {
  pipeline {
    respond {
      status = 200
      body = {
        environment = coalesce(env("APP_ENV"), "production")
      }
    }
  }
}
```
