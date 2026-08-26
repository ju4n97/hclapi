# Introduction

Hclapi is a language-agnostic, declarative API runtime designed to eliminate repetitive HTTP routing, input validation, and database mapping code.

## What is Hclapi?

Hclapi compiles declarative HCL manifests into an executable HTTP service. Instead of writing boilerplate backend code in Go, Python, or Node.js, endpoints are declared in configuration files alongside their validation rules and execution pipelines.

Hclapi operates as a standalone server daemon distributed as a single binary or container. For teams working in Go, it can also be embedded directly as a standard `http.Handler`.

## Core design principles

- **Language-agnostic operation:** Applications are declared entirely through HCL, Starlark, and SQL. No Go knowledge is required to build and run standalone APIs.
- **Explicit data flow:** Magic scopes are prohibited. Variables passing between HTTP inputs, Starlark scripts, and SQL queries must be bound explicitly through context paths (`ctx.request.body`, `steps.<name>.result`).
- **Compile-time safety:** Manifests are parsed into an Abstract Syntax Tree (AST) at startup. Missing references, invalid types, and unsafe SQL queries cause startup termination.
- **Deterministic sandboxing:** Data manipulation relies on Google's Starlark (Python dialect). Scripts run in isolated memory pools with zero host network or filesystem access.
