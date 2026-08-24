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

The command loads the full notebook and validates an existing, nonblank focus ID; a nonblank comma-separated set of canonical link types; a nonblank search query; a positive limit; and the required boolean `--json` flag. Diagnostics are opt-in through `--explain`, which also explicitly requires `--json`.

It performs directed BFS from the focus, following only stored source-to-target links whose types are allowed. Each adjacency list is ordered by target ID, type, and annotation. The first predecessor recorded for each reachable node therefore defines one deterministic minimum-hop witness.

Search scoring remains a full-corpus operation. Both scorer inputs are the complete note set in ID order. Positive scores are normalized by the largest positive score across all notes, including unreachable notes. Candidate destinations exclude the focus and require both reachability and positive direct lexical BM25 evidence in the destination's title, body, or tags; relevance contributed only by graph-derived fields such as link annotations does not make a destination eligible. Among eligible destinations, `relevance_score` and ranking retain the normalized full-corpus score. Candidates are ranked by normalized relevance descending, hop count ascending, and destination ID ascending before applying the limit.

Without `--explain`, JSON remains exactly the existing top-level array. Each entry has:

- `destination`: `id`, `title`, and `relevance_score`;
- `nodes`: the ordered focus-to-destination witness;
- `edges`: ordered `from`, `to`, `type`, and `annotation` values, where edge `i` joins node `i` to node `i+1`.

With `--explain`, JSON is the deterministic object `{routes:[existing route objects], diagnostics:{...}}`. Diagnostics are bounded aggregate metadata and contain no note bodies, titles, or candidate dumps:

- `query_tokens`: normalized tokenizer terms in deterministic query order;
- `total_notes`: notes in the scoring corpus;
- `direct_lexical_matches`: notes with positive title/body/tag BM25 evidence, excluding all link annotations;
- `focus_excluded`: 1 when the focus directly matched and was excluded from destinations, otherwise 0;
- `typed_reachable`: existing notes reachable through the selected typed directed links, excluding focus;
- `eligible_destinations`: the intersection of direct lexical matches and typed-reachable notes, excluding focus;
- `graph_scored_matches`: a deliberately separate count of positive full graph-aware relevance scores; it can exceed direct lexical matches because inbound or outbound annotations score, but those annotation-only scores do not create eligible routes;
- `returned`: route objects returned after limiting;
- `truncated_by_limit`: whether the limit removed at least one eligible ranked route.

Routes has no depth flag, so diagnostics do not invent a depth value. Reachability remains complete under the selected directed edge-type filter.

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
- Deterministic scoring input, tokenizer diagnostics, and predecessor selection make output stable across backend and stored-link ordering.
- Opt-in aggregate diagnostics explain tokenization, lexical eligibility, reachability elimination, annotation-aware scoring expansion, and limit truncation without exposing note content.
- The command returns one shortest witness per destination; it does not enumerate alternate paths or infer unstored relationships.
