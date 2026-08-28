# Why Hclapi

Hclapi was built for one specific situation, and may or may not generalize
beyond it.

## Origin

Several customers, each running a local SQL Server instance,
each needing a handful of read and write endpoints exposed publicly through
[FRP](https://github.com/fatedier/frp), since none of the databases are
reachable from outside the customer's network.

None of the individual endpoints are complicated. A lookup by ID, an insert
with a constraint check, a paginated list. The difficulty was mostly in
having to repeat this for every customer. A small backend service per
customer works, but it means a codebase, a build, and a deploy pipeline per
customer, for something that is, in each case, only a few endpoints wide.

Hclapi is an attempt to make that repetition cheaper. Each customer gets a
manifest file instead of a service. The queries sit directly in that file,
so understanding an endpoint doesn't require reading through a data access
layer, and changing one is closer to editing a config file than shipping a
release.

This worked reasonably well for that case. It has not been tested at scale,
across many teams, or against every kind of workload, and it's
possible parts of this design don't hold up outside the situation it came
from.

## What it tries to do

- **Keep endpoints readable.** A manifest is meant to be read start to
  finish and understood, without needing to know a framework first.
- **Keep changes cheap.** A query change is a file edit and a restart, not
  a rebuild.
- **Avoid a few common mistakes by default.** SQL parameters are bound
  rather than interpolated, and Starlark scripts don't have file system or
  network access. These are properties of the engine, not things a reviewer
  has to catch by hand, though that's a narrower guarantee than it might
  sound like, it only covers what happens inside Hclapi.
- **Stay small enough to hand off.** The intent is that someone other than
  the original author can look at a manifest and understand it.

## Where it probably doesn't fit

Hclapi isn't meant to compete with general-purpose backend frameworks, and
it's probably a poor fit once a project needs more than a thin layer over
data that already exists.

- **Business logic beyond Starlark and SQL.** A [`go` step](./steps/go.md)
  exists for this, but at that point Hclapi is mostly acting as a router in
  front of Go code, not replacing the need for it.
- **Schema management.** Hclapi doesn't have an ORM or migrations. It assumes
  something else already owns the schema.
- **Authentication.** Route level auth guards exist, but there's no OAuth
  flow, session handling, or identity provider built in. That's expected to
  live upstream, or in a [`go` step](./steps/go.md).
- **Running many services at once.** One manifest tree is one service.
  Whatever runs and manages multiple instances of it, systemd, Docker,
  Kubernetes, is outside Hclapi's concern.
- **Workflows with real branching or state.** A pipeline is a straight
  line. Anything that needs retries, loops, or long running state is
  probably better off as actual code.

If a project needs several of the above, Hclapi is likely the wrong choice,
and a general-purpose framework will serve it better. Hclapi is offered as a
small, fairly narrow tool that happened to work for one recurring problem,
not as a broader claim about how backends should be built.
