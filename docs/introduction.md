---
title: Introduction
description: A declarative backend engine that turns HCL manifests, SQL, and Starlark into HTTP APIs.
---

# hclapi

hclapi is a backend engine distributed as a single binary. It turns HashiCorp
Configuration Language (HCL) manifests into HTTP APIs, combining data access,
business logic, validation, and API definitions in a single declarative
configuration, with built-in OpenAPI generation.

:::note Request-time manifests

Manifests are read and executed at request time. hclapi does not generate or
compile Go source.

:::

## Getting started

:::card-group{cols="2"}

::card{title="Installation" icon="arrow-down-tray" href="/installation"}

Download the binary or install hclapi from source.

::

::card{title="Quickstart" icon="rocket-launch" href="/quickstart"}

Create a manifest, start the server, and call two HTTP endpoints.

::

:::

## Documentation

:::card-group{cols="3"}

::card{title="Concepts" icon="book-open" href="/concepts/lifecycle"}

The request lifecycle, execution context, pipeline model, expressions, and
error handling shared by every manifest.

::

::card{title="Manifest" icon="document-text" href="/manifest/structure"}

The HCL blocks used to define servers, connections, schemas, endpoints,
types, and functions.

::

::card{title="Pipeline steps" icon="adjustments-horizontal" href="/steps/sql"}

The step types available inside a pipeline, including SQL, Starlark, Redis,
transactions, parallel execution, Go, and responses.

::

::card{title="Guides" icon="book-open" href="/guides/go"}

Guides for embedding hclapi into a Go application and registering native Go
functions as pipeline steps.

::

::card{title="Patterns" icon="squares-2x2" href="/patterns"}

Recurring pipeline patterns referenced across endpoint configurations.

::

::card{title="CLI reference" icon="command-line" href="/cli/hclapi"}

Reference for the hclapi command-line interface.

::

:::

## Machine-readable documentation

Access the documentation in formats designed for AI assistants, agents, and developer tooling:

:::card-group{cols="2"}

::card{title="llms.txt" icon="document-text" href="/llms.txt"}

A compact index of the documentation for efficient retrieval.

::

::card{title="llms-full.txt" icon="document-text" href="/llms-full.txt"}

Full documentation context in a single text file.

::

:::

## How hclapi works

An endpoint consists of an HTTP route, optional request validation, and an
ordered pipeline of steps.

:::steps

1. A request is matched against an `endpoint`.

2. Path, query, header, and body data are validated when a request schema is
   defined.

3. Pipeline steps execute in order and write their outputs into the request
   context.

4. A `respond` step terminates the pipeline and writes the HTTP response.

:::

See [Request lifecycle](./concepts/lifecycle.md) for the complete execution
model and [Pipelines and steps](./concepts/pipelines.md) for pipeline
execution rules.

## What hclapi is for

hclapi is intended for small HTTP APIs where the API layer is mostly a thin
interface over existing data.

It keeps endpoint definitions, queries, validation, and request processing
close together in the manifest.

hclapi does not replace a general-purpose backend framework. Services that
require substantial application logic, complex workflows, long-running state,
identity and authentication, schema management, or multiple cooperating
services may be better implemented in application code.

## Source

Source and issue tracker:

[https://github.com/ju4n97/hclapi](https://github.com/ju4n97/hclapi)
