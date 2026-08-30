---
title: hclapi serve
---

<!-- This file is auto-generated. Do not edit by hand. -->

## NAME

serve - Start the hclapi HTTP server.

### SYNOPSIS

serve

```
[--config|-c|--manifests|-m]=[value]
[--host|-h]=[value]
[--port|-p]=[value]
[--verbose|-v]
```

**Usage**:

```
serve [GLOBAL OPTIONS] [command [COMMAND OPTIONS]] [ARGUMENTS...]
```

### GLOBAL OPTIONS

**--config, -c, --manifests, -m**="": Path to .hcl file, or directory containing manifests. (default: ".")

**--host, -h**="": Host address to bind the server (overrides manifest).

**--port, -p**="": Port to bind the server (overrides manifest). (default: 0)

**--verbose, -v**: Enable verbose debug logging.
