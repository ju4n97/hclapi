# Security policy

If you find a security vulnerability in `hclapi`, please report it privately. Don't open public issues for security vulnerabilities.

## Supported versions

`hclapi` uses trunk-based development and does not maintain previous release branches.

Security fixes are provided only for the latest release. Older releases are not maintained and should be upgraded to the latest version when a security issue is fixed.

## Reporting a vulnerability

The preferred way to report a vulnerability is through [GitHub Security Advisories](https://github.com/ju4n97/hclapi/security/advisories).

You can also contact the maintainer directly using the email address listed in the Git commit history.

Please include:

- A summary of the vulnerability.
- A minimal `.hcl` manifest, `curl` command, or other steps needed to reproduce it.
- The expected impact and the actual impact you observed.

`hclapi` is a free, open-source project and does not operate a bug bounty program. Security reports are appreciated, but no payment or other reward should be expected.

We will review security reports and, when appropriate, publish fixes along with security release notes.

## Security model

`hclapi` includes several protections by default:

- **SQL injection:** SQL queries use prepared statement parameters (`$1`, `?`, `@p1`). Raw string interpolation is not supported.
- **Denial of service (DoS):** Request bodies are limited with `http.MaxBytesReader` to prevent memory exhaustion from oversized payloads.
- **Sandboxed Starlark:** Starlark scripts run in an isolated memory sandbox with no network or filesystem access, and execution is strictly step-bounded.
- **Header injection (CRLF):** Dynamic header keys and values are sanitized to remove `\r` and `\n` before being written to the HTTP transport.
- **Information leakage:** Internal database connection strings, credentials, and raw stack traces are never exposed in client error responses.
