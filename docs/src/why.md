# Why hclapi

hclapi is for building small HTTP APIs over existing databases without writing a separate backend for each one.

An API is defined in a manifest, with queries and processing steps kept close to the endpoint that uses them. The result is a small service that is easy to read, change, and deploy.

hclapi is intentionally narrow. It works best when the API is mostly a thin layer over existing data.

## What it provides

- **Readable endpoints.** A manifest should be understandable without knowing a framework or data access layer.
- **Cheap changes.** Query and endpoint changes are file changes rather than application code changes.
- **Safe defaults.** SQL parameters are bound instead of interpolated, and Starlark cannot access the filesystem or network.
- **Small surface area.** The system is designed to stay simple enough to understand and maintain.

## Where it does not fit

hclapi does not replace a general-purpose backend framework.

Use application code when the service needs substantial business logic, complex workflows, or long-running state. Schema management, identity and authentication, and running multiple services are also outside hclapi's scope.

A `go` step is available when logic cannot reasonably live in SQL or Starlark, but at that point hclapi is primarily providing the API layer around Go code.

hclapi is best suited to small, data-oriented APIs where keeping the endpoint definition close to the query is more useful than introducing a larger application stack.
