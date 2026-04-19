# ADR-0008: Discovery and Traversal

**Status:** Accepted — pending implementation
**Date:** 2026-04-19
**Authors:** jaresty

---

## Context

`nn` has strong capture and organisation primitives but limited discovery and traversal.
BM25 search (`nn search`) retrieves notes for known queries. The link graph is navigable
via `nn show --linked-from` (one hop) and `nn path` (shortest path). `nn clusters` surfaces
topological groupings.

What is missing is *unknown-unknown* discovery — surfacing notes the user has forgotten, and
connections that exist in the graph but haven't been made explicit — and *multi-hop traversal*
for loading a coherent subgraph as context for an LLM or for review.

Two features address these gaps:

1. **`nn random`** — serendipitous discovery via a random note.
2. **`nn list --similar <id>`** — surface notes related to a given note by BM25 overlap, without
   requiring a known query.
3. **`nn show --depth N`** — extend `show` to traverse the link graph to depth N, collecting all
   reachable notes and printing them as a single Markdown document.

---

## Decisions

### 1. `nn random`

Return one randomly selected note. Composes with the existing `nn list` filters.

```
nn random
nn random --status permanent
nn random --tag philosophy
nn random --type concept
nn random --json
```

Implementation: single `ORDER BY RANDOM() LIMIT 1` query over the notes index, applying any
specified filter predicates. No index changes required.

`--json` returns the same shape as `nn list --json --rich` for a single note.

Use case: deliberate serendipity — re-encounter a note and consider whether it connects to
current work.

### 2. `nn list --similar <id>`

Return notes ranked by BM25 similarity to the note at `<id>`, excluding `<id>` itself.

```
nn list --similar <id>
nn list --similar <id> --limit 10
nn list --similar <id> --status permanent --json
```

Implementation: use the existing SQLite FTS5 index. Query the FTS table with the body text
of `<id>` as the query string, rank results by BM25 score, exclude `<id>`. No schema changes
required — the FTS index already contains full body text.

This is deliberately keyword-based (not embedding-based). It surfaces notes that share
vocabulary with the query note, which is sufficient for Zettelkasten-scale notebooks where
notes on the same topic naturally share terminology.

Composes with existing `nn list` flags: `--limit`, `--status`, `--tag`, `--type`, `--json`.

### 3. `nn show --depth N`

Extend `nn show` to traverse the link graph up to N hops from the given note ID, collect all
reachable notes, and print them in a single Markdown document separated by `---`.

```
nn show <id> --depth 2
nn show <id> --depth 3 --status permanent
nn show <id> --depth 2 --type concept --json
```

This extends the existing `--linked-from` flag behaviour (which shows direct outgoing links)
to arbitrary depth. `--linked-from` is implicitly `--depth 1` from the given note.

**Graph walk:** BFS from `<id>` using the in-memory link graph already constructed at query
time. Traversal respects the directionality of links (follows outgoing links). Filter flags
(`--status`, `--type`, `--tag`) apply per-note: notes that don't match are excluded from
output but their links are still followed (they are not cut points in the traversal).

**Output — plain Markdown (default):**
Each note is printed as its full Markdown content, prefixed with a `## <title> (<id>)` heading,
separated by `---`. The starting note appears first. No template system — the LLM or human
reader reformats as needed.

**Output — `--json`:** Returns an array of note objects in BFS order (same shape as
`nn show --json`), plus a `depth` field on each indicating how many hops from the origin.

**No separate `nn traverse` command.** Traversal is conceptually "show me this note and
its neighbourhood" — extending `show` is correct. A new command would be a synonym with no
added clarity.

**No templating.** The output is concatenated Markdown. When an LLM is in the loop it
reformats trivially; when a human uses it, raw Markdown is more readable than a templated
format they didn't configure.

**`--depth` default:** No default depth — `--depth` must be specified explicitly to avoid
accidentally printing the entire graph. `show` without `--depth` behaves as today.

---

## Implementation Order

1. `nn random` — trivial DB query, high delight-to-effort ratio ☐
2. `nn list --similar` — uses existing FTS index, no schema changes ☐
3. `nn show --depth` — BFS walk, reuses existing graph and show logic ☐
4. `nn status` suggested connections — pairs of unlinked notes with high similarity ☐
   *(depends on `--similar` being proven out first)*

---

## Alternatives Considered

**Embedding-based similarity (`nn similar` with vector search):** Rejected. Requires an
embedding model (external API or local binary), a vector column in the index, and index
rebuild on every note change. Incompatible with nn's single-binary, zero-external-dependency
philosophy. BM25 over shared vocabulary is sufficient at notebook scale. Can be revisited
if BM25 proves inadequate.

**Separate `nn traverse` command:** Rejected. Traversal is a depth-extended form of `show`.
Adding a command for it would split a coherent behaviour across two entry points with
overlapping flags.

**Templated output for `nn show --depth`:** Rejected. The LLM in the loop handles reformatting
trivially. A template system adds configuration surface area and implementation complexity for
no real benefit.

**Following links in both directions during traversal:** Deferred. BFS over outgoing links
covers the primary use case (loading context rooted at a topic). Bidirectional traversal
(following backlinks too) would be a `--bidirectional` flag added later if needed.
