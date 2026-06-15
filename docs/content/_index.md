---
title: "simbad"
description: "Query the SIMBAD Astronomical Database from the command line."
heroTitle: "simbad, from the command line"
heroLead: "Query millions of stars, galaxies, and nebulae in the SIMBAD Astronomical Database. One pure-Go binary, no API key, output that pipes into the rest of your tools."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`simbad` queries the SIMBAD Astronomical Database at simbad.u-strasbg.fr via
ADQL/TAP over plain HTTPS. No API key, nothing to run alongside it.

```bash
simbad query --type G --top 10       # galaxies
simbad stars --top 20                # stars
simbad object "M 31"                 # look up Andromeda by name
simbad tap "SELECT TOP 5 main_id,ra,dec,otype_txt FROM basic"  # raw ADQL
simbad serve --addr :7777            # the same operations over HTTP
```

Output adapts to where it goes: an aligned table on your terminal, JSONL the
moment you pipe it somewhere.

## Two ways to use it

- **As a command** for querying SIMBAD by hand or in a script. Start with
  the [quick start](/getting-started/quick-start/).
- **As a resource-URI driver** so a host like
  [ant](https://github.com/tamnd/ant) can address SIMBAD as
  `simbad://` URIs. See [resource URIs](/guides/resource-uris/).

Both are the same code: one operation, declared once, is a CLI command, an HTTP
route, an MCP tool, and a URI dereference.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/).
- Doing a specific job? The [guides](/guides/) are task-first.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
