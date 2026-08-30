---
title: starlark
description: Execute sandboxed Starlark for data transformation, filtering, and business logic.
---

# starlark

Executes sandboxed Starlark for data transformation, filtering, and business logic.

## Declaration

```hcl
starlark "<name>" {
  source = <<-STARLARK
    def execute(ctx):
      return { "status": "processed" }
  STARLARK
}
```

The script must define `execute(ctx)`, returning a dict, list, or scalar. `ctx` mirrors the [execution context](../concepts/context.md), using Starlark's dict access instead of dotted attributes.

## Attributes

| Attribute | Type     | Required | Description                             |
| :-------- | :------- | :------- | :-------------------------------------- |
| label     | `string` | yes      | step identifier                         |
| `source`  | `string` | yes      | Starlark source defining `execute(ctx)` |

## Examples

Default fallbacks and subscript lookups.

```python
def execute(ctx):
  prefix = ctx.request.body.get("prefix", "default_prefix")
  tags = ctx.request.body.get("tags", [])
  user_id = ctx.request.body["user_id"]  # raises if absent

  return { "prefix": prefix, "total_tags": len(tags), "user_id": user_id }
```

List comprehension.

```python
def execute(ctx):
  raw_tags = ctx.request.body.get("tags", [])
  prefix = ctx.request.body.get("prefix", "tag")

  cleaned = [prefix + ":" + t.strip().lower() for t in raw_tags if len(t.strip()) > 0]
  return { "count": len(cleaned), "tags": cleaned }
```

Reshaping prior step results.

```python
def execute(ctx):
  account = ctx.steps.fetch_account
  invoices = ctx.steps.fetch_invoices

  total_spent = sum([inv["amount_cents"] for inv in invoices if inv["status"] == "paid"])

  return {
    "account_id": account["id"],
    "name": account["name"].strip().title(),
    "total_spent_cents": total_spent
  }
```

## Sandboxing

Starlark scripts have no file system or network access. Iteration order and arithmetic are deterministic across platforms. Unbounded recursion and infinite loops are prevented by the runtime.
