# ADR-0009: Graph Query and Export

**Status:** Accepted — pending implementation
**Date:** 2026-04-19
**Authors:** jaresty

---

## Context

`nn` has strong capture, search, and traversal primitives, but no way to explore the *whole
picture* of a notebook's link graph. The existing tools are query-oriented (answer a specific
question) rather than browse-oriented (wander and discover):

- `nn show --depth N` loads a subgraph as Markdown context — useful for LLMs, not for humans
  wanting to see topology.
- `nn status` surfaces orphans as a side-effect of protocol checking.
- `nn path` finds the shortest path between two known notes.

What is missing:

1. **Graph queries** — "what are the most-linked notes?", "what notes act as bridges between
   clusters?", "what are the orphans?" — surfaced as first-class commands, not side-effects.
2. **Structured subgraph export** — LLM-readable JSON and human-readable DOT/SVG, so an LLM
   agent or a human can reason over the graph without loading every note's content.
3. **Self-contained visual export** — a single HTML file for browsing and sharing without a
   running server.

The TUI graph browser was considered and deferred: terminal graph layout is a hard UX problem,
the implementation cost is high, and the result is neither portable nor shareable.

---

## Decisions

### 1. `nn graph` subcommand group

All graph query and export operations live under `nn graph`. Existing `nn status` orphan
output is preserved as-is; `nn graph orphans` is a first-class alias that emits the same
set in machine-readable form.

### 2. Graph query commands

Three query commands are added to `internal/graph` and exposed via `nn graph`:

**`nn graph top [--limit N]`**
Returns notes ranked by inbound link count (degree centrality). Default limit: 10.
Output: plain text (one `id — title (N links)` per line) or `--format json`.
Index requirement: no schema change needed; query is `SELECT to_id, COUNT(*) AS n FROM links GROUP BY to_id ORDER BY n DESC LIMIT ?`.

**`nn graph orphans`**
Returns notes with no inbound and no outbound links.
Reuses the query already in `internal/index`; exposes it as a standalone command.
Output: same format as `nn graph top`.

**`nn graph bridges [--limit N]`**
Returns notes that connect otherwise disconnected parts of the graph — high-betweenness
approximation. V1 approximation: notes that appear on the most two-hop paths (i.e., notes N
where `count(distinct A) * count(distinct B) > threshold` for paths A→N→B). Full betweenness
centrality is deferred; the approximation is good enough for Zettelkasten scale.
Output: same format as `nn graph top`.

All query commands respect `--format json` for LLM consumption and pipe-friendliness.

### 3. JSON subgraph export

**`nn graph show [--focus <id>] [--depth N] [--format json|text]`**

Emits the ego-graph of a focal note as structured data. Default depth: 2. Default format: text
(same as `nn show --depth N` today). With `--format json`:

```json
{
  "center": "abc123",
  "nodes": [{"id": "abc123", "title": "...", "tags": [...]}],
  "edges": [{"from": "abc123", "to": "def456"}]
}
```

Node list is sorted by ID for deterministic diffs. This is the primary LLM-facing interface.
`CLAUDE.md` will document: "Use `nn graph show --focus <id> --depth 2 --format json` to
retrieve a subgraph for reasoning."

Without `--focus`, emits the full graph (all notes + all links). Useful for small notebooks;
at scale the caller should always specify `--focus`.

### 4. DOT and SVG export

**`nn graph export [--format dot|svg] [--focus <id>] [--depth N] [--open]`**

- `--format dot`: emits Graphviz DOT syntax. No external dependency; pure text serialization.
  Caller is responsible for rendering (e.g., `nn graph export --format dot | dot -Tpng`).
- `--format svg`: shells out to `dot -Tsvg` if `graphviz` is on PATH; otherwise errors with
  a helpful message. Does not bundle a Go SVG layout library — the external dep is acceptable
  for this optional render path.
- `--open`: after writing to a temp file, opens in the default viewer (`open` on macOS,
  `xdg-open` on Linux). Only valid with `--format svg` or `--format html`.
- Scoping via `--focus` and `--depth` works identically to `nn graph show`.

### 5. Self-contained HTML export (stretch)

**`nn graph export --format html [--output path]`**

Writes a single `.html` file containing:
- Embedded D3 v7 force-layout (pinned version, fetched from CDN at build time and stored in
  `templates/graph.html` via `embed.FS`).
- The full graph JSON inline in a `<script>` tag — no server required.
- Basic UI: click a node to highlight its neighbours; hover shows title and tags.

No server is ever started. The file is self-contained and shareable. This is a stretch goal;
it is not a blocker for Phases 1–3.

### 6. No TUI graph browser

A Bubble Tea graph browser was considered. It is deferred indefinitely:
- Terminal graph layout (force-directed in a fixed character grid) is a hard UX problem.
- Output is not portable or shareable.
- Implementation cost (L–XL) is disproportionate to the browsability gain.

If this is revisited, it should be a separate ADR.

### 7. Index schema

No schema changes are required for Phases 1–3. All query commands derive from the existing
`links` table. If thematic clustering (tag-based or embedding-based similarity) is added later,
it may require a new table or view; that decision is deferred.

---

## Implementation Sequence

1. **Phase 1 — Query layer**: `nn graph top`, `nn graph orphans`, `nn graph bridges`
2. **Phase 2 — JSON subgraph**: `nn graph show --format json`
3. **Phase 3 — DOT export**: `nn graph export --format dot` (SVG as optional follow-on)
4. **Phase 4 (stretch) — HTML export**: `nn graph export --format html`

Each phase is independently shippable. Phase 2 is most valuable to LLM-assisted workflows.
Phase 3 is most valuable for human spot-checking. Phase 4 is most valuable for sharing.

---

## Consequences

- `nn graph` becomes the canonical entry point for all graph-level operations, distinguishing
  them from note-level operations (`nn show`, `nn list`, etc.).
- LLM agents gain a stable, deterministic JSON interface for subgraph reasoning.
- Humans gain a DOT/SVG export path with no new required runtime dependencies.
- The orphan concept is now surfaced in two places (`nn status` and `nn graph orphans`);
  both are kept for their respective audiences.
- Thematic clustering (`nn graph clusters`) is explicitly out of scope for this ADR.
