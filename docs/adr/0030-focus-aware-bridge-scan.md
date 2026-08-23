# ADR 0030: Bridge Scan excludes retained positions before limiting

## Status

Accepted

## Context

An integrated navigation trial ran query-conditioned bridge Scan from the highest-relevance consultant-workflow focus. The current focus was itself the first bridge result. Although structurally valid, it was not a movement candidate. Filtering it in presentation after the command's limit could also return fewer alternatives than requested.

## Decision

`nn graph bridges --search QUERY --format json` accepts repeatable `--exclude ID` flags.

The command computes bridge eligibility and structural scores over the complete graph, computes query relevance, and ranks candidates exactly as before. It then removes excluded IDs and finally applies `--limit`:

```text
limit(rank(full-graph bridge candidates) − excluded IDs)
```

Blank exclusion IDs are errors. Unknown nonblank IDs are harmless. `--exclude` requires `--search`; bridge behavior and output remain unchanged when the flag is absent.

Navigation Scan passes the retained focus as an exclusion.

## Consequences

- Scan offers movement candidates rather than rediscovering its current position.
- Excluded high-ranking candidates are replaced up to the requested limit.
- Repeated exclusions can suppress other already-inspected positions without changing topology or relevance scoring.
