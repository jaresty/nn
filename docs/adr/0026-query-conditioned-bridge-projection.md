# ADR 0026: Bridge search projects query relevance onto full-graph connectors

## Status

Accepted

## Context

`nn graph bridges` ranks connector notes globally using the existing heuristic `distinct inbound neighbors × distinct outbound neighbors`. During a goal-directed Scan, the global top ten can be dominated by unrelated hubs or aggregation notes. Computing bridges only from search hits would discard the nonmatching neighbors that establish a connector's structural role and could misrepresent global topology.

## Decision

Add query-conditioned bridge projection:

```text
nn graph bridges --search <query> --format json [--limit N]
```

The command first computes bridge candidates and structural scores over the complete graph exactly as legacy bridge output does. It independently runs normal full-corpus search over all notes. Search mode intersects positive-scoring search hits with the global bridge candidates; it does not compute the connector heuristic over a filtered graph.

Search results rank by normalized query relevance descending, structural bridge score descending, then note ID ascending. The limit is applied after filtering and ranking. Search JSON preserves `id`, `title`, and structural `score`, and adds `relevance_score`.

`--search` requires `--format json` and a non-blank query. Without `--search`, existing text and JSON behavior remain unchanged.

## Consequences

- Goal-directed Scan can request relevant highways without loading unrelated global bridges.
- A bridge remains eligible when its inbound and outbound neighbors do not match the query, because topology is computed globally.
- Matching notes that are not global bridge candidates are excluded.
- Returned note IDs are executable navigation handles for Peek or Recenter.
- This slice does not identify connected regions, assign stable region IDs, or claim graph-theoretic articulation points; those remain separate decisions.
