# ADR 0033: Impact traversal makes relationship direction explicit

## Status

Accepted

## Context

Typed destination discovery combines search relevance with directed reachability, and typed path finds one witness to a known destination. Neither answers the bounded impact-set question: “starting here, which notes are reached through this relationship family, in this chosen direction, and what shortest witness explains each result?” Datalog can derive closure, but it does not provide this command's ordered, depth-bounded predecessor witnesses or make reverse traversal explicit at the CLI boundary.

Relationship direction is especially important for evidence. A stored `grounded-by` edge points from a claim to its evidence, while a stored `supports` edge points from corroborating evidence to the supported claim. An impact query must be able to walk either with or against stored direction without rewriting the evidence carried by a witness.

## Decision

Add a structured graph query:

```text
nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json
```

All five flags are explicit and required. The command validates a nonblank focus ID that exists in the fully loaded notebook; a nonblank comma-separated set of canonical link types with no empty members; direction exactly `incoming` or `outgoing`; a positive depth; and a true boolean `--json` flag. The output reports `links` as a sorted unique array and preserves the requested depth.

Traversal is cycle-safe breadth-first search bounded by the requested depth:

- `outgoing` follows allowed stored edges from source to target;
- `incoming` builds the reverse adjacency and traverses from stored target to stored source.

Adjacency is deterministic by traversed endpoint ID, then edge type, then annotation. The first predecessor recorded for a note defines one deterministic shortest path. Missing stored targets are not traversable notes.

Every witness edge always retains stored source/from to target/to orientation and its annotation. Consequently, in incoming mode the `nodes` path is ordered from focus to impact, but each corresponding `edges` item points opposite the consecutive traversal node order. Reversing adjacency never reverses or synthesizes stored edge facts.

JSON is one object with:

- `focus`: `id` and `title`;
- `direction`, normalized sorted unique `links`, and requested `depth`;
- `impacts`: all reached notes except the focus, sorted by depth ascending then note ID ascending.

Each impact contains `node` (`id`, `title`), its BFS `depth`, ordered focus-to-impact `nodes`, and the stored-orientation `edges` witness (`from`, `to`, `type`, `annotation`). An empty impact set is encoded as `[]`.

## Evidence-direction examples

For constitutive evidence stored as:

```text
claim --grounded-by--> observation
```

`--focus observation --links grounded-by --direction incoming` traverses observation→claim. Its node witness has that traversal order, while its edge remains `claim` from/stored source → `observation` target, opposite the consecutive nodes.

For corroboration stored as:

```text
observation --supports--> claim
```

`--focus observation --links supports --direction outgoing` traverses observation→claim and the witness edge points in the same order because traversal follows the stored source-to-target direction.

## Navigation integration

Impact traversal is a bounded structural overlay for a retained focus:

- Scan reads the complete depth-bounded impact set under the selected relationship types and direction.
- Peek inspects one impact's `nodes` and `edges` witness without changing focus, remembering that incoming witness edges point opposite traversal order.
- Recenter advances to `nodes[1]` and reruns the query from the new focus rather than jumping to a distant impact.
- Arrive explains the traversed relationship types and annotations when focus reaches the selected impact.

## Consequences

- Callers must state semantic direction, depth, relationship family, and output contract instead of inheriting traversal defaults.
- Incoming analysis can answer “what relies on this?” while preserving the stored facts needed to explain why.
- Cycle safety, deterministic adjacency, result ordering, and one-shortest-predecessor policy make output stable without enumerating alternate paths.
- The command reports explicit stored relationships only; it does not infer unstored links, rank by semantic relevance, or replace Datalog closure queries.
- Existing graph, route, path, and show commands retain their behavior.
