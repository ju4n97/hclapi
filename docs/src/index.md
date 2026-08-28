# Hclapi

Hclapi is a declarative backend engine distributed as a single binary. It reads
HCL manifests, SQL queries, and Starlark scripts, and serves them as HTTP
APIs.

Manifests are read and executed at request time. Hclapi does not generate or
compile Go source.

See [Installation](./installation.md) to get the binary and
[Quickstart](./quickstart.md) to serve a working endpoint.

[Concepts](./concepts/README.md) describes the request lifecycle, the
execution context, and the pipeline model shared by every manifest.
[Manifest](./manifest/README.md) and [Pipeline steps](./steps/README.md) are
reference documentation for HCL block syntax. [Go integration](./go/README.md)
describes embedding Hclapi as a library inside a Go application.

Source and issue tracker: <https://github.com/ju4n97/hclapi>.
