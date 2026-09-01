# Contributing to hclapi

Thanks for helping build `hclapi`. This codebase is kept simple, fast, and easy to maintain. It tries to follow most of the principles described in [Effective Go](https://go.dev/doc/effective_go).

## Documentation

Full project documentation is available at [ju4n97.github.io/hclapi](https://ju4n97.github.io/hclapi/).

The documentation is written in [MDX](https://mdxjs.com/), using [Rspress](https://rspress.rs/).

It's recommended that you read this documentation before contributing to have full context and understanding of this project.

## How the engine works

1. **Startup:** When `hclapi serve` starts, it walks the manifest directory, parses `.hcl` files into an AST, initializes database connection pools, and binds routes to Go's standard `http.ServeMux`.
2. **Request time:** Each incoming HTTP request creates an isolated context and executes pipeline steps sequentially (`sql`, `starlark`, `go`, etc.) until a `respond` step sends the HTTP response.

## Project structure

```text
cmd/hclapi/                     CLI entrypoint and subcommands (serve, docs)
hclapi.go                       Public library API for embedding in Go
internal/
  core/                         Domain models (Context, Server, Connection, ProblemDetailsError)
  parser/                       HCL manifest parsing and AST definitions
  eval/                         HCL expression evaluation and built-in functions
  connectors/connsql/           Database connection pools, drivers, and dialects
  steps/                        Individual step runners (xgo, xstarlark, xsql, xrespond)
  engine/                       Route dispatcher and pipeline execution runner
```

## Key engineering rules

- A single, cross-compilation binary is maintained with zero CGO dependencies. All database drivers must be pure Go.
- Avoid implicit fallbacks or hidden state. All steps export data under explicit keys (`.rows`, `.row`, `.value`, `.result`, etc.).
- Never mutate shared request state across step handlers.
- Invalid syntax, malformed durations, or broken connection URLs must fail immediately on startup with clear error messages, never during customer requests.

## Git workflow

`hclapi` uses trunk-based development. Keep changes small and focused, and avoid long-lived branches.

Each pull request should contain one logical change. Don't mix unrelated refactors, formatting changes, dependency updates, or other cleanup into a PR unless they are part of the same change.

Keep commits focused and make commit messages describe what changed. Don't add unnecessary generated text or long explanations to commit messages.

Merge commits are fine.

## Development workflow

This project uses [Taskfile](https://taskfile.dev) to manage common tasks:

```bash
# Run linters
task lint

# Run fast unit tests (in-memory SQLite, no Docker needed)
task test

# Run integration tests against real databases (requires Docker)
task test-integration
```

## Using AI

AI tools can be used when contributing to `hclapi`, but they're not a substitute for understanding the project or making technical decisions.

Use them if they're useful to you. You can use them to write code, tests, investigate a problem, or explore possible solutions and validate ideas. Anything you contribute must still be reviewed and understood by you. Don't blindly submit generated code, and don't treat an LLM's output as an authority.

For issues, comments, and PR descriptions, it's preferable to use your own words. Clear and direct is better than a generated summary that says more than necessary.

For project documentation, don't use AI to generate large amounts of text just for the sake of having more documentation. Documentation should be written from an actual understanding of the project and reviewed for correctness before it is contributed. Generated documentation that is inaccurate, redundant, or adds little value will likely be removed.

PRs that are considered fully vibe-coded will likely be closed as well.

## Project configuration

Keep the repository focused on the project itself. Don't add configuration that primarily exists to support your personal development environment, editor, AI tool, or workflow.

This includes things such as `.vscode/`, `.zed/`, `.cursor/`, `CLAUDE.md`, `AGENTS.md`, or similar files.

Add project configuration only when it provides a clear benefit to hclapi and its contributors as a whole.

## License

By contributing to `hclapi`, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
