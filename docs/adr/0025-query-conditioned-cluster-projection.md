# ADR 0025: Cluster search projects query evidence onto the full graph

## Status

Accepted

## Context

Navigation Teleport currently searches for relevant notes and separately requests every global cluster. The agent must correlate those outputs manually, producing large responses and weakly grounded landing-zone recommendations.

Clustering only the search hits would be smaller, but it would discard the surrounding topology that makes a region meaningful. Existing `nn clusters` output is also a public contract and should not change for callers that do not search.

## Decision

Add optional query-conditioned projection:

```text
nn clusters --search <query> --json
```

The command always runs deterministic label propagation over the complete graph first. Search then scores all notes against the full corpus, ordering scorer inputs by note ID so tied search evidence is independent of backend list order. Search mode returns only full-graph clusters containing at least one positive-scoring note; it never reclusters the hit-induced subgraph.

`--search` requires `--json` and a non-blank query. `--singletons` lowers the implicit minimum cluster size to one; an explicitly supplied `--min` remains authoritative.

For each returned region:

- `size` is full cluster size;
- `match_count` is the number of positive-scoring members;
- `match_density` is `match_count / size`, an explanatory signal, not a ranking input;
- `score` is the sum of the three strongest normalized member search scores (or all matches when fewer than three exist);
- `matches` lists matching members with scores;
- `notes` lists every full-cluster member;
- `representative` is the highest-total-degree member, breaking ties by note ID.

Regions sort by score descending, then representative ID ascending. Matches sort by score descending, then note ID ascending. Bounding the aggregate at three lets corroborating strong matches matter without allowing a giant region to win through an unbounded count of weak matches. `match_count` and `matches` still report every positive hit. Existing text and JSON output remain unchanged when `--search` is absent.

Search mode also accepts `--summary`. Summary output omits only the full `notes` array while retaining `size`, `match_count`, `match_density`, `score`, `representative`, and ranked `matches`. The representative ID is the region's navigation handle: Teleport recenters on it and re-enters Orient. `--summary` requires both `--search` and `--json`; ordinary search JSON remains unchanged without it.

## Consequences

- Teleport can request compact relevant landing regions directly instead of correlating a search hit list with every cluster or loading every member of large matching clusters.
- Region membership remains grounded in global topology.
- Large clusters are not rewarded merely for size or an unbounded accumulation of weak matches; only their three strongest matching-note scores contribute to ranking.
- The first slice retains single-membership label propagation. Overlapping, hierarchical, and stable region identities remain future decisions informed by observed navigation quality.
