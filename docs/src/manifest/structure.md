# Files and merging

Hclapi builds its runtime tree from a single file, or by walking a directory
and merging every manifest it finds into one service definition.

## Recognized files

| Pattern    | Example                          |
| :--------- | :------------------------------- |
| `Hclapifile` | `Hclapifile`, `routes/v1/Hclapifile` |
| `*.hcl`    | `main.hcl`, `connections.hcl`    |
| `*.hclapi`   | `api.hclapi`, `orders.hclapi`        |

Non-manifest files (`README.md`, `init.sql`, `.gitignore`, static assets)
are ignored during discovery.

## Directory scanning

When passed a directory, `hclapi serve -c ./config`, the parser walks the
tree recursively. Directories beginning with a dot (`.git`, `.cache`) are
skipped. All endpoints, connections, schemas, and server configurations
found are merged into a single AST.

## Merge rules

**Endpoints** are identified by method and path. A method-path pair declared
in more than one file halts startup with a diagnostic naming both files.

**Server blocks** merge by attribute. If more than one file declares
`server { }`, the last one evaluated takes precedence for any attribute it
sets explicitly. Unset attributes retain their defaults.

**Connections and schemas** occupy a global namespace. A connection labeled
`connection "postgres" "primary"` in `connections.hcl` is accessible from
any endpoint in the tree, as is any `schema` label.

## Layouts

A flat layout suits a single-file service.

```text
my-service/
├── Hclapifile
└── docker-compose.yaml
```

```hcl
server {
  host = "0.0.0.0"
  port = 8080
}

connection "postgres" "main" {
  url = env("DATABASE_URL")
}

endpoint "GET /health" {
  pipeline {
    respond {
      status = 200
      body   = { status = "OK" }
    }
  }
}
```

A domain-driven layout separates connections, schemas, and routes.

```text
api-service/
├── server.hcl
├── connections.hcl
├── schemas/
│   ├── account.hcl
│   └── user.hcl
└── routes/
    ├── accounts.hcl
    └── users.hcl
```

A versioned layout separates routes by API release.

```text
gateway/
├── Hclapifile
├── schemas/
│   ├── v1.hcl
│   └── v2.hcl
└── routes/
    ├── v1/
    └── v2/
```

`hclapi serve -c ./gateway` merges all versioned routes into a single router.
