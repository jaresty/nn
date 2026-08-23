# ADR 0027: Typed paths are directed navigation witnesses

## Status

Accepted

## Context

`nn path` returns a shortest undirected node sequence. That is useful for topological connection, but a path may mix or reverse relationship semantics and legacy JSON cannot explain which edges were traversed. Datalog can derive typed transitive closure, but the current rules interface returns facts rather than an ordered shortest witness that Navigation can walk.

## Decision

Add an optional typed mode:

```text
nn path <from> <to> --links <type,...> [--json]
```

When `--links` is present, traversal follows stored source-to-target orientation and admits only the listed known link types. BFS returns a minimum-hop witness under those constraints. Adjacency is ordered by target ID, link type, and annotation so equal-length witness selection is deterministic.

Typed JSON is an object with ordered `nodes` and `edges`. Every edge includes `from`, `to`, `type`, and `annotation`, and edge `i` connects node `i` to node `i+1`. Typed text retains the familiar node route. Empty filters and unknown link types fail.

Without `--links`, existing undirected text and JSON-array behavior remain unchanged.

## Navigation integration

Typed paths are a route overlay rather than a new navigation action:

- Orient computes a route when focus and destination are known.
- Teleport remains instant relocation; a typed route is an optional explainable alternative.
- Scan assesses semantically coherent routes to candidates.
- Peek previews the witness without moving.
- Recenter advances to the next node.
- Arrive explains the traversed edge sequence.

Datalog remains the appropriate mechanism for closure and impact sets. Typed path supplies the concrete ordered witness for “show me how.”

## Consequences

- Navigation can walk and explain semantic routes one edge at a time.
- Reverse-only or semantically disallowed routes are correctly absent in typed mode.
- Existing callers retain their prior output contract.
- This slice does not add all-path enumeration, weighted paths, Datalog provenance, or impact queries.
