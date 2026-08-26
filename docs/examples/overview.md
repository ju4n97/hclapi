# Examples catalog

The repository includes runnable, self-contained reference architectures under the `examples/` directory.

## Reference suites

* [01. Zero dependency](/examples/zero-dependency): In-memory execution and Starlark payload transformations without an external database.
* [02. SQLite CRUD](/examples/sqlite-crud): Single-file SQLite database operations with input schema validation and conditional 404 responses.
* [03. PostgreSQL transactions](/examples/postgres-transactions): Atomic multi-table writes with PostgreSQL unique constraint error mapping (`catch "23505"`).
* [04. Redis caching](/examples/redis-caching): Cache-aside implementation using Redis, TTL expiration, and early-return fast paths.
* [05. Parallel execution](/examples/parallel-execution): Concurrent database query fan-out for dashboard aggregation.
* [06. Go embedded plugin](/examples/go-embedded): Outbound HTTP requests and Go SDK integrations using native Go pipeline steps.
* [07. Modular production](/examples/modular-production): Multi-file directory architecture with primary/replica connection pooling and JWT authentication.
