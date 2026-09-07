# Contributing to hclapi

Thanks for contributing to `hclapi`. This project follows standard [Effective Go](https://go.dev/doc/effective_go) idioms to keep the codebase simple, fast, and easy to maintain.

It's recommended that you read the [documentation](https://ju4n97.github.io/hclapi/) before contributing for full context.

## How the Engine Works

1. **Boot time:** `hclapi serve` walks manifest directories, parses `.hcl` files into an AST, validates constraints, initializes database pools, compiles OpenAPI specs, and mounts routes to standard `http.ServeMux`.
2. **Request time:** Each HTTP request initializes an isolated `ExecutionContext`. Pipeline steps (`sqldb`, `star`, native Go) run sequentially until terminated by a `respond` step or an error.

## Project Structure

```text
hclapi/
├── cmd/hclapi/        # CLI: serve, openapi, version
├── hclapi.go          # Public Go library facade
└── internal/
    ├── config/        # Unified AST, HCL parsing, and boot validation
    ├── engine/        # HTTP routing, StepRegistry, and pipeline execution
    ├── sqldb/         # Multi-dialect SQL pooling, placeholders, and execution
    ├── star/          # Sandboxed Starlark script execution
    ├── openapi/       # OpenAPI 3.1 compilation and embedded UI renderers
    ├── problem/       # RFC 9457 Problem Details formatting
    ├── runtime/       # Request execution context and state
    ├── validator/     # Ingress schema and payload validation
    ├── eval/          # Expression evaluation and built-in functions
    └── scalar/        # Duration and ByteSize types
```

## Architecture and dependency rules

Imports must form a strict, unidirectional directed acyclic graph:

$$\text{cmd} \longrightarrow \text{engine} \longrightarrow \text{config} \longrightarrow \text{eval} \longrightarrow \text{runtime} \longrightarrow \text{problem}$$

- Leaf packages (`problem`, `scalar`, `star`, `sqldb`, `runtime`) must not depend on higher-level packages like `config` or `engine`.
- Nothing under `internal/` can import the root package `github.com/ju4n97/hclapi`.
- Dynamic request execution (`runtime`) must never import static manifest parsing (`config`).

## Key engineering rules

- Cross-compilation must work cleanly for all platforms. All database drivers must be pure Go.
- Syntax errors, invalid durations, missing connections, or conflicting routes must halt startup immediately with actionable diagnostics; never during request handling.
- No hidden state or implicit fallbacks. Steps always export data under explicit keys (`.rows`, `.row`, `.result`).
- Never mutate shared request state across step handlers.

## Development workflow

This project uses [Taskfile](https://taskfile.dev) for common tasks:

```bash
task lint              # Run linters
task fmt               # Format code and docs
task test              # Fast unit tests (SQLite in-memory)
task test-race         # Run tests with the Go race detector
task test-integration  # Run integration tests against real services (Docker required)
task build             # Compile local binary
task build-all         # Cross-compile for all release targets (requires goreleaser)
```

## Git workflow

This project uses trunk-based development with small, focused pull requests into `main`.

Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat(config): add support for custom scalar units`
- `fix(sqldb): handle sqlite primary key collision`
- `chore: update dependencies`

**Breaking changes:** Append an exclamation mark (`!`) before the colon in the subject (e.g. `feat(config)!: move openapi to root block`) and explain the migration steps under a `BREAKING CHANGE:` footer. This is needed for automated changelog generation.

## Release process

Releases are automated via GitHub Actions and [GoReleaser](https://goreleaser.com) adhering to [SemVer](https://semver.org/).

1. **Verify `main`:**

   ```bash
   task lint && task test && task test-race && task test-integration
   ```

2. **Tag and push:**

   ```bash
   task tag -- v0.2.0   
   ```

## Guidelines on AI tools and workspace config

- AI tools are allowed. However, you're responsible for testing, understanding, and verifying that contributions meet the architecture standards.
- Keep descriptions and commit messages direct and concise. Avoid pasting large AI generated summaries.
- Don't commit personal editor or AI tooling configurations (`.vscode/`, `.zed/`, `.cursor/`, `CLAUDE.md`, etc.).

## License

By contributing, you agree that your code will be licensed under the project's [MIT License](LICENSE).
