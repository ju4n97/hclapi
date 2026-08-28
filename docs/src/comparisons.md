# Comparisons

Hclapi overlaps with several existing tools without matching any of them exactly. The difference is mostly in what each tool assumes about the database, API surface, and surrounding infrastructure.

## PostgREST

PostgREST generates a REST API directly from a Postgres schema. There is no manifest to maintain; the API surface comes from tables, views, roles, and RLS policies.

That works well when the API should closely mirror the database. The tradeoff is that schema changes also change the API.

Hclapi takes the opposite approach: every endpoint is an explicit pipeline. This means more to write up front and no automatic schema sync, but the API exposes exactly what is written and nothing more.

PostgREST is also Postgres-only. Hclapi's `sql` step is driver-agnostic in
principle.

## Hasura / PostGraphile

Hasura and PostGraphile generate GraphQL APIs from a database schema, with relationships, permissions, and subscriptions built in.

They solve a broader problem than Hclapi. Hclapi does not provide GraphQL, subscriptions, or automatic relationship resolution. For those requirements, Hasura or PostGraphile are the better fit. Hclapi is aimed at a smaller case: a handful of explicit REST endpoints with no real-time requirements.

## Supabase

Supabase is a full backend platform: Postgres, auth, storage, realtime, edge functions, and a generated API.

Hclapi intentionally provides none of that. It assumes the database and infrastructure already exist and adds a thin HTTP layer over a small part of it. Supabase makes more sense for a complete backend; Hclapi makes more sense when only a narrow API layer is needed.

## Low-code workflow tools (n8n, and similar)

These tools focus on orchestrating systems, often through events and visual workflows.

Hclapi's pipelines are superficially similar, but they are text-based, tied to individual HTTP endpoints, and deliberately avoid branching, retries, and long-running workflows. Once the problem becomes multi-system orchestration, these tools are a better fit.

## Writing a backend service directly

For problems with substantial business logic, writing a backend service may still be the right choice. Hclapi is not meant to replace an application; it is for cases where writing one feels disproportionate to what needs to be exposed.

## Summary

|                | Hclapi                               | PostgREST                | Hasura / PostGraphile               | Supabase              | n8n-like                             |
| :------------- | :--------------------------------- | :----------------------- | :---------------------------------- | :-------------------- | :----------------------------------- |
| API surface    | explicit, hand-written             | derived from schema      | derived from schema                 | derived from schema   | visual workflow                      |
| Query language | SQL, written by hand               | none needed              | GraphQL, generated                  | mixed                 | none                                 |
| Scope          | thin HTTP layer over existing data | REST over Postgres       | GraphQL over a database             | full backend platform | cross-system orchestration           |
| Auth           | none built in                      | Postgres roles / RLS     | own permission system               | built in              | varies                               |
| Best fit       | small, isolated endpoint sets      | API should mirror schema | relational, real-time GraphQL needs | full backend platform | event-driven, multi-system workflows |
