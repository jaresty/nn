# ADR 0032: Typed destination discovery combines relevance with directed reachability

## Status

Accepted

## Context

Navigation can find a relevant note with corpus search and can produce a typed shortest witness once both endpoints are known. It cannot ask a positioned question such as “which relevant evidence destinations can I actually reach from here through `grounded-by` links?” A global ranked hit may be disconnected, reachable only in reverse, or reachable only through semantically inappropriate edge types.

## Decision

Add a structured graph query:

```text
nn graph routes --focus ID --links TYPES --search QUERY --limit N --json
```

The command loads the full notebook and validates an existing, nonblank focus ID; a nonblank comma-separated set of canonical link types; a nonblank search query; a positive limit; and the required boolean `--json` flag.

It performs directed BFS from the focus, following only stored source-to-target links whose types are allowed. Each adjacency list is ordered by target ID, type, and annotation. The first predecessor recorded for each reachable node therefore defines one deterministic minimum-hop witness.

Search scoring remains a full-corpus operation. Both scorer inputs are the complete note set in ID order. Positive scores are normalized by the largest positive score across all notes, including unreachable notes. Candidate destinations exclude the focus and require both reachability and positive direct lexical BM25 evidence in the destination's title, body, or tags; relevance contributed only by graph-derived fields such as link annotations does not make a destination eligible. Among eligible destinations, `relevance_score` and ranking retain the normalized full-corpus score. Candidates are ranked by normalized relevance descending, hop count ascending, and destination ID ascending before applying the limit.

JSON is an array. Each entry has:

- `destination`: `id`, `title`, and `relevance_score`;
- `nodes`: the ordered focus-to-destination witness;
- `edges`: ordered `from`, `to`, `type`, and `annotation` values, where edge `i` joins node `i` to node `i+1`.

## Navigation integration

Typed destination discovery is the route-selection layer when focus and goal are known but destination is not:

- Orient presents ranked reachable destinations.
- Scan assesses reachable query-relevant territory under a semantic filter.
- Peek inspects a witness without moving.
- Recenter advances to `nodes[1]`.
- Arrive explains the traversed witness and treats relevance as discovery evidence, not edge strength.

`nn path --links` remains the direct route query when the destination is already known. Datalog remains the closure and impact-set mechanism.

## Consequences

- Search results offered as moves are guaranteed to have a typed directed witness from the retained focus.
- Reverse-only, disallowed-type, unreachable, and notes without positive direct lexical evidence do not become destinations.
- Deterministic scoring input and predecessor selection make output stable across backend and stored-link ordering.
- The command returns one shortest witness per destination; it does not enumerate alternate paths or infer unstored relationships.
