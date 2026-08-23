# ADR 0029: Search-mode bridges include deterministic crossing witnesses

## Status

Accepted

## Context

`nn graph bridges --search QUERY --format json` identifies relevant load-bearing notes using a full-graph `incoming-neighbor count × outgoing-neighbor count` heuristic. The result previously exposed structural and relevance scores but did not explain what concrete edges made a returned note bridge-like. Scan therefore supplied navigation handles without enough local evidence to distinguish useful crossings.

## Decision

Search-mode bridge JSON includes one incoming and one outgoing witness edge:

```text
incoming endpoint ─edge→ bridge ─edge→ outgoing endpoint
```

Each endpoint contains its note ID and title. Each edge contains its type and annotation. For each side, selection uses the lexicographically smallest `(endpoint ID, edge type, annotation)` tuple. This makes output independent of backend note order and stored edge order.

Witness construction occurs only after full-graph bridge eligibility, structural scoring, relevance ranking, and limit application. It does not alter those operations.

Legacy bridge text and JSON remain unchanged when `--search` is absent.

## Consequences

- Scan can explain one concrete reason a returned note is a plausible crossing before Peek or Recenter.
- The witness is a deterministic crossing example, not proof that its endpoints belong to distinct graph territories.
- One edge per side keeps search JSON compact; deeper neighborhood or path inspection remains the role of Peek, Orient, and typed paths.
