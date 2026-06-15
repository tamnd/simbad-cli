---
title: "Introduction"
description: "What simbad is and how it is put together."
weight: 10
---

A command line for the SIMBAD Astronomical Database.

`simbad` is a single binary. It speaks to simbad.u-strasbg.fr over plain HTTPS
via the ADQL/TAP protocol, shapes the responses into clean records, and gets out
of your way. There is nothing to sign up for and nothing to run alongside it.

## What you can do

- **query** — filter astronomical objects by type (G=Galaxy, *=Star, SNR=Supernova Remnant, etc.)
- **object** — look up any object by name or identifier (M31, Andromeda, NGC 224, alpha Centauri)
- **stars** — list stars from the catalog
- **tap** — execute a raw ADQL SELECT statement for full flexibility

## How it is built

- A **library package** (`simbad`) holds the HTTP client and the typed
  data models. It paces requests, sets an honest User-Agent, and retries the
  transient failures any public service throws under load.
- A **domain** (`simbad/domain.go`) declares each operation once on the
  [any-cli/kit](https://github.com/tamnd/any-cli) framework. That single
  declaration becomes a CLI command, an HTTP route, an MCP tool, and a
  resource-URI dereference.
- A thin **`cmd/simbad`** hands the assembled app to `kit.Run`.

## One operation, four surfaces

Because an operation is surface-neutral, the same query you run on the command
line is also a route and a tool:

```bash
simbad query --type G              # the command
simbad serve --addr :7777          # GET /v1/query?type=G
simbad mcp                         # the query tool, over stdio
```

You write the fetch and the record shape; the surfaces come for free.

## Scope

`simbad` is a read-only client over data SIMBAD already serves publicly. It
reads that data and shapes it for you. That narrow scope keeps it a single small
binary with no database, no daemon, and no setup.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
