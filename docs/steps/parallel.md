# Parallel step

The `parallel` block executes child steps concurrently across worker goroutines.

## Concurrency model

```mermaid
flowchart LR
    Start[Pipeline Flow] --> Fork{Fork Goroutines}
    Fork --> StepA[sql: fetch_profile]
    Fork --> StepB[sql: fetch_orders]
    Fork --> StepC[sql: fetch_metrics]
    StepA --> Join[sync.WaitGroup Barrier]
    StepB --> Join
    StepC --> Join
    Join --> Merge[Merge Results into ctx.Steps]
    Merge --> Next[Next Pipeline Step]
```

## Syntax

```hcl
parallel {
  sql "fetch_profile" {
    connection = connection.postgres.main
    query      = "SELECT * FROM users WHERE id = @id"
    args       = { id = ctx.request.path.id }
  }

  sql "fetch_orders" {
    connection = connection.postgres.main
    query      = "SELECT * FROM orders WHERE user_id = @id LIMIT 5"
    args       = { id = ctx.request.path.id }
  }

  sql "fetch_notifications" {
    connection = connection.postgres.main
    query      = "SELECT * FROM notifications WHERE user_id = @id LIMIT 10"
    args       = { id = ctx.request.path.id }
  }
}
```
