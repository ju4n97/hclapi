# Go integration

hclapi embeds into an existing Go application as a library. It mounts native
handlers alongside declarative routes and supports calling back into Go
from a pipeline step.

See [Embedding](./embedding.md) to mount hclapi inside a Go service, or
[Registering steps](./registering-steps.md) to register a function for a
`go` step.
