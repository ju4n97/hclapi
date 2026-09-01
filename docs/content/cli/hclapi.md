---
title: hclapi
---

<!-- This file is auto-generated. Do not edit by hand. -->

## NAME

hclapi - Declarative API runtime that turns HCL manifests into structured HTTP services.

### SYNOPSIS

hclapi

**Usage**:

```
hclapi [GLOBAL OPTIONS] [command [COMMAND OPTIONS]] [ARGUMENTS...]
```

### COMMANDS

#### serve

Start the hclapi HTTP server.

**--config, -c, --manifests, -m**="": Path to .hcl file, or directory containing manifests. (default: ".")

**--host, -h**="": Host address to bind the server (overrides manifest).

**--port, -p**="": Port to bind the server (overrides manifest). (default: 0)

**--verbose, -v**: Enable verbose debug logging.

#### openapi

Export the compiled OpenAPI 3.1 specification for your manifests.

**--config, -c, --manifests, -m**="": Path to .hcl file or directory containing manifests. (default: ".")

**--format, -f**="": Output format: json or yaml. (default: "json")

**--output, -o**="": Path to output file (defaults to stdout).

**--pretty**: Pretty-print JSON output.

#### version, v

Show detailed version information.
