# CLI reference

Command line tooling for serving manifests and running static analysis.

## hclapi serve

Starts the standalone HTTP daemon.

```sh
hclapi serve [options]
```

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--config` | `-c` | `.` | Path to manifest directory or single `Hclapifile` |
| `--verbose` | `-v` | `false` | Enable debug-level structured logging |

## hclapi validate

Statically validates manifest syntax, references, and schema bindings without starting the server.

```sh
hclapi validate [path]
```
