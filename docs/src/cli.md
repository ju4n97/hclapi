# CLI Reference

<!-- This file is auto-generated. Do not edit by hand. -->

## CLI interface - hclapi

Declarative API runtime that turns hclapi manifests into structured HTTP services.

Usage:

```bash
hclapi [COMMAND] [COMMAND FLAGS] [ARGUMENTS...]
```

### `serve` command

Starts the hclapi HTTP server.

Usage:

```bash
hclapi [GLOBAL FLAGS] serve [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                                       | Description                                          | Type   | Default value |        Environment variables        |
| ------------------------------------------ | ---------------------------------------------------- | ------ | :-----------: | :---------------------------------: |
| `--config="…"` (`-c`, `--manifests`, `-m`) | Path to .hcl file, or directory containing manifests | string |     `"."`     | `HCLAPI_CONFIG`, `HCLAPI_MANIFESTS` |
| `--host="…"` (`-h`)                        | Host address to bind the server (overrides manifest) | string |               |        `HCLAPI_HOST`, `HOST`        |
| `--port="…"` (`-p`)                        | Port to bind the server (overrides manifest)         | int    |      `0`      |        `HCLAPI_PORT`, `PORT`        |
| `--verbose` (`-v`)                         | Enable verbose debug logging                         | bool   |    `false`    |          `HCLAPI_VERBOSE`           |
