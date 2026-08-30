---
title: Execution context
description: The request-scoped ctx object and the data exposed by requests and completed pipeline steps.
---

# Execution context

`ctx` is the request-scoped object every step reads from. It holds the incoming request and the accumulated output of prior steps.

```text
ctx
├── timestamp_epoch : int
├── request
│   ├── method  : string
│   ├── path    : map[string]string
│   ├── query   : map[string]string
│   ├── headers : map[string]string
│   └── body    : any
└── steps
    ├── <sql_step_name>
    │   ├── rows          : list(map)
    │   ├── row           : map (or null)
    │   └── rows_affected : int
    ├── <redis_step_name>
    │   └── value         : any
    ├── <starlark_step_name>
    │   └── result        : any
    └── <go_step_name>
        └── result        : any
```

## Request

Populated once, at ingress. Immutable for the life of the request.

| Field                 | Type                | Example                             |
| :-------------------- | :------------------ | :---------------------------------- |
| `ctx.request.method`  | `string`            | `"POST"`                            |
| `ctx.request.path`    | `map[string]string` | `ctx.request.path.id`               |
| `ctx.request.query`   | `map[string]string` | `ctx.request.query.page`            |
| `ctx.request.headers` | `map[string]string` | `ctx.request.headers.authorization` |
| `ctx.request.body`    | `any`               | `ctx.request.body.email`            |

Path parameters are bound from route templates: `endpoint "GET /accounts/{id}"` binds `{id}` to `ctx.request.path.id`. Catch-all parameters like `{filepath...}` bind the remaining path to `ctx.request.path.filepath`. Header keys are lowercased at ingress.

## Steps

Populated incrementally as each step completes. Each step exports attributes tailored to its domain:

| Step type      | Exported fields                                                           | Description                                                                                                                                                            |
| :------------- | :------------------------------------------------------------------------ | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`sql`**      | `steps.<name>.rows`<br>`steps.<name>.row`<br>`steps.<name>.rows_affected` | • `rows`: List of all returned rows (`[]` if none)<br>• `row`: The first row (`null` if none)<br>• `rows_affected`: Total rows returned, inserted, updated, or deleted |
| **`redis`**    | `steps.<name>.value`                                                      | The retrieved cache value, or `null` on a cache miss                                                                                                                   |
| **`starlark`** | `steps.<name>.result`                                                     | The returned value from `def execute(ctx)`                                                                                                                             |
| **`go`**       | `steps.<name>.result`                                                     | The returned value from the registered Go function                                                                                                                     |

## Access from HCL and Starlark

In HCL expressions, `steps` is available as a root-level shorthand for `ctx.steps`. In Starlark scripts, all values are accessed under `ctx` using dictionary syntax.

| Value              | HCL expression                      | Starlark script                                  |
| :----------------- | :---------------------------------- | :----------------------------------------------- |
| Path parameter     | `ctx.request.path.id`               | `ctx.request.path["id"]`                         |
| Query parameter    | `ctx.request.query.filter`          | `ctx.request.query.get("filter")`                |
| Header             | `ctx.request.headers.authorization` | `ctx.request.headers.get("authorization")`       |
| Body field         | `ctx.request.body.name`             | `ctx.request.body["name"]`                       |
| SQL single record  | `steps.lookup.row.user_id`          | `ctx.steps.lookup.get("row", {}).get("user_id")` |
| SQL record list    | `steps.list_users.rows`             | `ctx.steps.list_users["rows"]`                   |
| Redis cache value  | `steps.cache_lookup.value`          | `ctx.steps.cache_lookup.get("value")`            |
| Starlark/Go result | `steps.compute.result`              | `ctx.steps.compute["result"]`                    |
| Ingress timestamp  | `ctx.timestamp_epoch`               | `ctx.timestamp_epoch`                            |
