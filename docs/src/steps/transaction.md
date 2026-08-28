# transaction

Groups multiple `sql` steps into one atomic transaction. A failure in any
nested query rolls back the entire block.

## Declaration

```hcl
transaction "<name>" {
  connection = connection.<driver>.<name>

  sql "step_one" {
    # ...
  }

  sql "step_two" {
    # ...
  }
}
```

## Attributes

| Attribute | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| label | `string` | yes | transaction identifier |
| `connection` | `connection` | yes | pool to acquire the transaction handle from |
| `sql` blocks | `block` | at least one | executed sequentially inside the transaction |

Steps within the block may reference each other's results in the normal
way.

## Rollback semantics

1. A query error triggers an immediate `ROLLBACK`.
2. A `catch` block that calls `abort_with_status` also rolls back, and
   writes its error payload directly to the client.
3. If every statement succeeds, the engine issues `COMMIT` before continuing
   to the next pipeline step.

See [Transactional writes](../patterns.md#transactional-writes) for a
complete example.
