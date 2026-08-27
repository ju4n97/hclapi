---
title: starlark (Scripting)
description: In-memory Python-dialect transformations and logic execution.
---

# Starlark

The `starlark` step executes deterministic, sandboxed business logic using Starlark. Starlark steps handle data transformation, list filtering, payload normalization, and custom business calculations with sub-millisecond execution times.

## Block declaration

A `starlark` step requires a unique label and a `source` string defining an `execute(ctx)` entry point:

```hcl
starlark "<name>" {
  source = <<-STARLARK
    def execute(ctx):
      # Transformation logic here
      return {
        "status": "processed"
      }
  STARLARK
}
```

## Attribute reference

| Attribute | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| **`name`** (Label) | `string` | Yes | Unique step identifier used under `steps.<name>.result`. |
| **`source`** | `string` | Yes | The Starlark source code. Must define an `execute(ctx)` function. |

## The `execute(ctx)` entrypoint

Every Starlark script must define a function named `execute` that accepts a single argument `ctx`.

```python
def execute(ctx):
  # Reads from ctx.request or ctx.steps
  # Returns a dictionary, list, or scalar
  return {"result": True}
```

### Context topology inside Starlark

The `ctx` parameter provides access to the full execution context using standard Starlark structures:

* `ctx.request.method`: String HTTP verb.
* `ctx.request.path`: Dictionary of path variables (e.g. `ctx.request.path["id"]`).
* `ctx.request.query`: Dictionary of query string parameters (e.g. `ctx.request.query.get("page", "1")`).
* `ctx.request.headers`: Dictionary of lowercase request headers (e.g. `ctx.request.headers.get("authorization")`).
* `ctx.request.body`: Dictionary or list representing the unmarshaled JSON body.
* `ctx.steps.<name>`: Dictionary or list representing the output of preceding steps.
* `ctx.timestamp_epoch`: Integer Unix timestamp.

## Standard operations

### 1. Dictionary lookups and default fallbacks

Dynamic JSON maps and headers are standard Starlark dictionaries, supporting `.get(key, default)` and subscript lookups:

```python
def execute(ctx):
  prefix = ctx.request.body.get("prefix", "default_prefix")
  tags = ctx.request.body.get("tags", [])
  user_id = ctx.request.body["user_id"] # Subscript lookup (fails if key is missing)
  
  return {
    "prefix": prefix,
    "total_tags": len(tags),
    "user_id": user_id
  }
```

### 2. List comprehensions and filtering

```python
def execute(ctx):
  raw_tags = ctx.request.body.get("tags", [])
  prefix = ctx.request.body.get("prefix", "tag")
  
  cleaned = [prefix + ":" + t.strip().lower() for t in raw_tags if len(t.strip()) > 0]
  
  return {
    "count": len(cleaned),
    "tags": cleaned
  }
```

### 3. Aggregating and reshaping previous step results

```python
def execute(ctx):
  account = ctx.steps.fetch_account
  invoices = ctx.steps.fetch_invoices
  
  total_spent = sum([inv["amount_cents"] for inv in invoices if inv["status"] == "paid"])
  
  return {
    "account_id": account["id"],
    "name": account["name"].strip().title(),
    "paid_invoices_count": len([inv for inv in invoices if inv["status"] == "paid"]),
    "total_spent_cents": total_spent
  }
```

## Sandboxing and security guarantees

Starlark is designed for strict hermetic isolation:

* **No file system or network access**: Scripts cannot make external network calls, open sockets, or access the disk.
* **Deterministic execution**: Iteration order and arithmetic behavior are deterministic across operating systems.
* **Bounded execution**: Unbounded recursion and infinite loops are prevented by the Starlark runtime engine.
