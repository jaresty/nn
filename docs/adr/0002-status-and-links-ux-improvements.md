# ADR-0002: Status and Links UX Improvements

**Status:** Accepted — implemented
**Date:** 2026-04-12
**Authors:** jaresty

**Implementation log:**
- 2026-04-12 All three decisions implemented: `nn status` inline orphan names, `nn status --json`, `nn links <id>` with `--json`

---

## Context

During real-world use of `nn` with a 26-note notebook, three usability gaps were identified:

1. **`nn status` reports orphan count but not orphan identities.** The output `orphans: 1`
   is not actionable — the user must grep the notebook directory or write a script to find
   which note is the orphan. The count is only useful once you know which note it is.

2. **No way to view link annotations in bulk.** `nn link` requires an annotation on every
   link, but there is no command that surfaces those annotations. Annotations are visible in
   the raw note body (`## Links` section) but not via any structured query. This makes the
   link graph opaque: you can see that two notes are connected but not why.

3. **`nn status --json` errors instead of producing JSON.** The `--json` flag is documented
   as cross-cutting (all commands), but `nn status` does not implement it. This violates the
   contract established in ADR-0001 §5 and breaks LLM-driven status checks.

---

## Decisions

### 1. `nn status` names orphans inline

`nn status` output changes from:

```
orphans: 1
```

to:

```
orphans: 1
  20260410-1234  The Overlooked Insight
```

In `--json` mode (see decision 3), orphans become an array of objects:

```json
{
  "orphans": [
    { "id": "20260410-1234", "title": "The Overlooked Insight" }
  ]
}
```

The count is preserved as `len(orphans)`. No separate `orphan_count` field is needed.

### 2. `nn links <id>` shows outgoing links with annotations

A new subcommand `nn links <id>` prints every outgoing link from the given note, including
the annotation for each:

```
20260409-5678  The Foundational Principle
  provides the foundational philosophy this principle implements

20260410-9012  The Contradicting View
  contradicts: argues atomicity is context-dependent
```

With `--json`:

```json
[
  {
    "id": "20260409-5678",
    "title": "The Foundational Principle",
    "annotation": "provides the foundational philosophy this principle implements"
  }
]
```

A `--backlinks` flag (or separate invocation) is out of scope for this ADR; the immediate
need is annotation visibility on outgoing links.

**Note on `nn graph`:** `nn graph` already outputs the full link graph as JSON or Graphviz
dot, but it does not include annotations and is scoped to the whole notebook. `nn links <id>`
is per-note and annotation-first.

### 3. `nn status` implements `--json`

`nn status --json` outputs a JSON object covering all fields currently in the human-readable
output:

```json
{
  "total": 26,
  "by_status": {
    "draft": 24,
    "reviewed": 1,
    "permanent": 1
  },
  "by_type": {
    "concept": 10,
    "argument": 6,
    "model": 4,
    "hypothesis": 3,
    "observation": 2,
    "question": 1
  },
  "orphans": [
    { "id": "20260410-1234", "title": "The Overlooked Insight" }
  ],
  "broken_links": []
}
```

This completes the `--json` cross-cutting contract for all commands.

---

## Consequences

- `nn status` becomes actionable without secondary scripting
- Annotations are auditable via a structured command, preserving the Zettelkasten invariant
  that connections carry meaning
- LLM-driven workflows can use `nn status --json` for programmatic health checks
- `nn links <id>` enables annotation review as a first-class operation

---

## Alternatives Considered

**`nn list --orphan` instead of inline orphan names in `nn status`:** `nn list --orphan`
already exists and returns orphan notes. The status output change is additive — a user
running `nn status` gets the answer immediately without a second command. Both remain useful.

**Annotations in `nn graph` output:** `nn graph` is graph-first (topology); adding
annotations would change its contract. A dedicated `nn links` command is cleaner and
composable with `--json`.
