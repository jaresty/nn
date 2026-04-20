# ADR-0010: Link Suggestion, Gap Analysis, and Weekly Review

**Status:** Accepted — pending implementation
**Date:** 2026-04-20
**Authors:** jaresty

---

## Context

`nn` is an LLM-driven Zettelkasten. Notes are captured and linked manually or via `nn link`,
but the notebook's value compounds with link density — the more connections exist, the more
the graph surfaces non-obvious relationships. At low note counts (< 100) this is manageable
manually. At higher counts it becomes intractable.

The gap is not in capture or search — those are well-served — but in three related workflows
that currently require the LLM to do all the work ad hoc:

1. **Link suggestion**: after creating or updating a note, discovering which existing notes it
   should connect to and with what relationship type.
2. **Gap analysis**: given a topic, understanding what the notebook covers thoroughly, what it
   covers shallowly, and what is entirely absent.
3. **Review**: a periodic health report that surfaces structural improvements across the whole
   notebook.

A study of similar LLM-Zettelkasten workflows (e.g. Obsidian + Claude via MCP) shows that
automated link suggestion with ~60-70% acceptance rates is the primary driver of link density
growth and the compounding effect that makes a Zettelkasten useful at scale.

---

## Decisions

### 1. `nn suggest-links <id>`

Produces a ranked list of candidate connections between the given note and existing notes in
the notebook. Each candidate includes:

- Source and target note IDs and titles
- Suggested link type (one of nn's canonical types: refines, contradicts, supports, extends,
  questions, source-of)
- A one-sentence rationale

The command loads the focal note's content and a structured summary of all other notes
(id, title, type, tags, first sentence of body) and emits a prompt-ready output for the LLM
to reason over. It does **not** call an LLM directly — it formats context for the LLM session
in which it is invoked, consistent with nn's design as a tool used by an LLM, not an LLM itself.

Output format: plain text list (default) or `--format json` for programmatic consumption.

```
nn suggest-links <id> [--limit N] [--format json]
```

The LLM receiving this output is expected to suggest links and then call `nn link` or
`nn bulk-link` to create the accepted ones. The typical workflow is:

```
nn suggest-links <id>          # load context
# LLM reasons over it and suggests links
nn link <id> <target> --type <type> --annotation "..."  # per accepted suggestion
```

### 2. `nn gap <topic>`

Loads all notes matching `<topic>` (via BM25 search, same as `nn search`) plus all notes
in the same graph neighborhood (backlinks and forward links) and formats them as a structured
context block prompting the LLM to identify:

- What is thoroughly covered
- What is mentioned but shallow (linked but thin)
- What is entirely absent
- Questions the current notes raise but don't answer

This is a context-loading command, not an analysis command — the LLM performs the analysis
from the formatted output.

```
nn gap <topic> [--depth N] [--limit N] [--format json]
```

Default depth: 1 (direct neighbors of search results). Default limit: 20 notes.

### 3. `nn review`

Runs a structured notebook health report combining existing commands and formatting the
output for LLM-driven analysis. Components:

1. **Growth**: total notes, notes by type, notes created in last 7/30 days
2. **Connectivity**: average links per note, notes with zero links (orphans), notes with
   only outbound links (dead-ends — no inbound)
3. **Clusters**: top clusters by size (from `nn clusters`)
4. **Hubs**: top notes by degree (from `nn graph top`)
5. **Structural gaps**: orphans, bridges, notes with draft status

Output is a single formatted Markdown block ready to paste into an LLM session for
interpretation and recommendations.

```
nn review [--format json]
```

### 4. Output format: context blocks, not analysis

All three commands follow the same principle as the rest of `nn`: they format context for
an LLM to reason over, rather than performing LLM reasoning themselves. This keeps the
commands fast, deterministic, and composable. The LLM in the session reads the output and
acts on it.

This is different from a hypothetical `nn suggest-links --auto` that would make API calls —
that may come later but is explicitly out of scope for this ADR.

### 5. `nn suggest-links` context format

The context block emitted by `nn suggest-links <id>` includes:

```markdown
## Focal note
id: <id>
title: <title>
type: <type>
tags: <tags>
body:
<full body>

## Candidate notes (N total)
### <id> — <title> [<type>]
tags: <tags>
summary: <first 200 chars of body>
existing links: <links to/from focal note if any>

...
```

Candidate notes are ranked by BM25 similarity to the focal note (most similar first) with a
cutoff at `--limit` (default 20). Notes already linked to the focal note are included but
marked as already linked, so the LLM can suggest additional link types or annotation
improvements.

### 6. `nn index <topic>`

Loads all notes matching `<topic>` (via BM25 + cluster detection), groups them into 3-5
conceptual clusters, and formats the result as a ready-to-review draft index note (Map of
Content). The LLM receiving this output is expected to name the clusters, identify tensions
and gaps, and create the index note via `nn new`.

```
nn index <topic> [--limit N] [--format json]
```

This builds on `nn clusters` (which already does label propagation) but scopes it to a
topic subset and formats for index-note generation rather than structural analysis.

Deferred to a later phase — depends on `nn gap` being useful first to understand what
topics warrant an index note.

### 7. `nn capture`

Streamlines the inbox→literature→permanent pipeline for processing external material
(articles, quotes, observations). Accepts raw text or a URL stub and creates a draft note
of type `observation` or `concept` pre-populated with:

- Title derived from the first line or URL
- Body with the raw content
- Status: draft
- Tags: empty (LLM fills these in the session)

```
nn capture [--title "..."] [--type observation|concept] --content "..."
```

This is essentially `nn new` with ergonomic defaults for capture workflows. The LLM then
refines the note, extracts atomic sub-notes via `nn new`, and runs `nn suggest-links` on each.

Deferred — low implementation complexity but depends on `nn suggest-links` being useful
first to complete the capture→link loop.

### 8. Dead-end detection in `nn review`

A "dead-end" note has outgoing links but no incoming links — it points to other notes but
nothing points back to it. This is distinct from an orphan (no links in either direction).
Dead-ends indicate notes that contribute to others but aren't integrated into the broader
graph. `nn review` surfaces both orphans and dead-ends.

---

## Implementation Sequence

1. **Phase 1**: `nn suggest-links <id>` — highest leverage, most immediate value
2. **Phase 2**: `nn review` — depends on existing commands, low implementation cost
3. **Phase 3**: `nn gap <topic>` — depends on search + graph traversal
4. **Phase 4**: `nn index <topic>` — depends on gap analysis being useful first
5. **Phase 5**: `nn capture` — depends on suggest-links completing the capture→link loop

Each phase is independently shippable.

---

## Consequences

- LLM-assisted sessions gain a structured entry point for link discovery after note creation.
- The weekly review workflow becomes a single command rather than manual assembly.
- Gap analysis enables directed note creation rather than ad-hoc capture.
- No LLM API calls are made by nn itself — the LLM consuming these commands performs all
  reasoning, consistent with the existing architecture.
- `nn index <topic>` and `nn capture` are natural follow-ons scoped to later phases.
- `nn contradictions` (detecting conflicting claims across unlinked notes) is deferred
  beyond this ADR — it requires either embedding-based similarity or a direct LLM reasoning
  step over note pairs, neither of which fits the context-loading pattern cleanly.
