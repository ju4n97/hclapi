# Starlark step

The `starlark` step executes deterministic, sandboxed Python-dialect scripts for payload transformation.

## Syntax

```hcl
starlark "normalize" {
  source = <<-STARLARK
    def execute(ctx):
        body = ctx.request.body
        tags = [t.strip().lower() for t in body.tags if len(t.strip()) > 0]
        return {
            "name": body.name.strip(),
            "email": body.email.strip().lower(),
            "tags": tags
        }
  STARLARK
}
```

## Execution semantics

* **Mandatory entrypoint:** Scripts must define an `execute(ctx)` function.
* **Context access:** Dot notation is enabled for dictionary navigation (`ctx.request.body.name`).
* **Isolation guarantees:** Starlark executes with zero access to host I/O, network sockets, or OS system calls.
* **Return handling:** The return value is serialized into `ctx.Steps[<step_name>].Result`.
