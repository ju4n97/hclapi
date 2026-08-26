# Why Hclapi & limitations

An architectural overview of where Hclapi fits, where it excels, and when it is the wrong tool for the job.

## Motivation & scope

Hclapi is not an all-encompassing backend framework, an ORM, or a full application server. It does not attempt to replace general-purpose programming languages, event-driven microservices, or complex domain-driven architectures.

Hclapi is a specialized, declarative utility built to solve one specific problem: reducing the boilerplate of straightforward data APIs and I/O workflows.

## Where Hclapi is a good fit

* **Data access APIs:** Exposing clean, parameterized CRUD endpoints directly over PostgreSQL, SQLite, MySQL, or MSSQL.
* **Internal tools & ops endpoints:** Lightweight administrative backends, reporting endpoints, and webhook receivers.
* **Rapid prototyping & proof-of-concepts:** Standing up production-grade HTTP contracts with live validation and OpenAPI documentation before committing to a custom application stack.
* **Read-through caching layers:** Implementing cache-aside workflows (Redis <-> SQL) with minimal configuration.
* **Hybrid Go applications:** Embedding Hclapi inside an existing Go service to offload repetitive CRUD endpoints while keeping heavy business logic in native Go.

## Limitations & non-goals

### 1. Complex domain-driven design (DDD)

If an application requires deep aggregate roots, rich in-memory domain behavior, state machines with dozens of business transitions, or distributed sagas, Hclapi is the wrong tool. Hand-crafted code in Go, Rust, or Java using clean or hexagonal architecture is strictly superior for complex domain logic.

### 2. Microsecond low-latency systems

While Hclapi's state machine and Starlark runtime are fast and memory-pooled, systems requiring sub-millisecond, zero-allocation buffer management (e.g., high-frequency trading or real-time packet processing) should be written in pure Go or C.

### 3. Dynamic runtime query construction

Hclapi enforces static, parameterized SQL queries compiled at startup to ensure SQL injection safety. It is not an arbitrary query builder for runtime dynamic filtering across arbitrary user-selected columns with variable `WHERE` clauses (unless handled via custom Go extensions).

### 4. Full-stack BaaS features

Hclapi does not include built-in multi-tenant user management tables, real-time WebSocket replication, or object storage proxies. It provides the API runtime layer; surrounding identity, storage, and network infrastructure remain the responsibility of the operator.
