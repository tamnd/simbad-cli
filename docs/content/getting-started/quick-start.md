---
title: "Quick start"
description: "Query your first astronomical objects with simbad."
weight: 30
---

Once `simbad` is on your `PATH`, query some galaxies:

```bash
simbad query --type G --top 5
```

You get an aligned table by default. Ask for JSON when you want to pipe it:

```bash
$ simbad query --type G --top 2 -o json
[
  {
    "main_id": "M 31",
    "ra": 10.684,
    "dec": 41.269,
    "object_type": "G"
  },
  ...
]
```

## Look up a specific object

```bash
simbad object "M 31"
simbad object "Andromeda"
simbad object "alpha Centauri"
simbad object "NGC 224"
```

The identifier is matched via SIMBAD's full alias table, so common names, Messier
numbers, NGC numbers, and catalog identifiers all work.

## List stars

```bash
simbad stars --top 20
```

## Run a raw ADQL query

For full flexibility, pass any ADQL SELECT statement directly:

```bash
simbad tap "SELECT TOP 10 main_id,ra,dec,otype_txt FROM basic WHERE otype_txt='SNR' ORDER BY main_id"
```

## Shape the output

The same flags work on every command:

```bash
simbad query --type G --fields main_id,ra,dec   # keep only these columns
simbad object "M 31" -o jsonl | jq .redshift    # into jq
simbad stars --top 100 -o csv > stars.csv       # export to CSV
```

`-o` takes `table`, `json`, `jsonl`, `csv`, `tsv`, `url`, or `raw`. Left to
`auto`, it prints a table to a terminal and JSONL into a pipe. See
[output formats](/reference/output/) for the full contract.

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
simbad serve --addr :7777 &
curl -s 'localhost:7777/v1/query?type=G&top=5'   # NDJSON
simbad mcp                                        # MCP over stdio
```
