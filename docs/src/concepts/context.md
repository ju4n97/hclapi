# Execution context

`ctx` is the request-scoped object every step reads from. It holds the
incoming request and the accumulated output of prior steps.

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
    └── <step_name>
        ├── result        : any
        └── rows_affected : int
```

## request

Populated once, at ingress. Immutable for the life of the request.

| Field                 | Type                | Example                             |
| :-------------------- | :------------------ | :---------------------------------- |
| `ctx.request.method`  | `string`            | `"POST"`                            |
| `ctx.request.path`    | `map[string]string` | `ctx.request.path.id`               |
| `ctx.request.query`   | `map[string]string` | `ctx.request.query.page`            |
| `ctx.request.headers` | `map[string]string` | `ctx.request.headers.authorization` |
| `ctx.request.body`    | `any`               | `ctx.request.body.email`            |

Path parameters are bound from the route template: `endpoint "GET /accounts/{id}"`
binds `{id}` to `ctx.request.path.id`. Header keys are lowercased at ingress.

## steps

Populated incrementally as each step completes. A step may read any prior
step's output. A step may not modify a prior step's output.

| Field                            | Type  | Set by     |
| :------------------------------- | :---- | :--------- |
| `ctx.steps.<name>.result`        | `any` | every step |
| `ctx.steps.<name>.rows_affected` | `int` | `sql` only |

## Access from HCL and Starlark

In HCL expressions, `steps` is available as a root-level shorthand for
`ctx.steps`. In a [`starlark`](../steps/starlark.md) script, all values are
nested under `ctx`, and map access uses Starlark's dict syntax.

| Value           | HCL                                 | Starlark                                   |
| :-------------- | :---------------------------------- | :----------------------------------------- |
| Path parameter  | `ctx.request.path.id`               | `ctx.request.path["id"]`                   |
| Query parameter | `ctx.request.query.filter`          | `ctx.request.query.get("filter")`          |
| Header          | `ctx.request.headers.authorization` | `ctx.request.headers.get("authorization")` |
| Body field      | `ctx.request.body.name`             | `ctx.request.body["name"]`                 |
| Step result     | `steps.lookup.result.user_id`       | `ctx.steps.lookup["user_id"]`              |
| Timestamp       | `ctx.timestamp_epoch`               | `ctx.timestamp_epoch`                      |

`.get(key, default)` returns `default` if the key is absent. Subscript
access raises an error if the key is absent.

Go step handlers receive an equivalent structure through `*hclapi.Context`.
See [Registering steps](../go/registering-steps.md#context-data-access).
