---
title: Pipelines and steps
description: The ordered step model inside endpoints, including execution rules, outputs, and failure behavior.
---

# Pipelines and steps

A pipeline is an ordered sequence of steps declared inside an `endpoint` block. Each step performs one unit of work and writes its result into the execution context for subsequent steps to read.

## Step catalog

| Step                                   | Block declaration      | Function                                      | Exported outputs                                                          |
| :------------------------------------- | :--------------------- | :-------------------------------------------- | :------------------------------------------------------------------------ |
| [sql](../steps/sql.md)                 | `sql "<name>"`         | Parameterized database queries and mutations  | `steps.<name>.rows`<br>`steps.<name>.row`<br>`steps.<name>.rows_affected` |
| [redis](../steps/redis.md)             | `redis "<name>"`       | Cache reads, writes, and key deletions        | `steps.<name>.value`                                                      |
| [starlark](../steps/starlark.md)       | `starlark "<name>"`    | Sandboxed Python-like data transformation     | `steps.<name>.result`                                                     |
| [go](../steps/go.md)                   | `go "<name>"`          | Invokes a registered native Go function       | `steps.<name>.result`                                                     |
| [transaction](../steps/transaction.md) | `transaction "<name>"` | Atomic multi-statement SQL execution          | Step outputs of nested `sql` blocks                                       |
| [parallel](../steps/parallel.md)       | `parallel { }`         | Concurrent branch execution                   | Step outputs of nested branch steps                                       |
| [respond](../steps/respond.md)         | `respond { }`          | Terminates the pipeline with an HTTP response | None (Terminal step)                                                      |

## Step execution rules

1. **Unique step labels:** Every step (except `respond` and `parallel`) requires a unique name label. The label defines its namespace under `steps.<name>`.
2. **Read-only context history:** A step may read any output written by prior steps, but cannot modify prior outputs or mutate `ctx.request`.
3. **Fail-fast errors:** An unhandled error in any step (a database query failure, Starlark step limit exceeded, or Go panic) immediately halts the pipeline and triggers the error handler.
4. **Early termination:** The pipeline terminates at the first `respond` step whose `condition` evaluates to `true`, or at the first unconditional `respond`.
