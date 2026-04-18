# ADR-0007: Graph Intelligence and Code Awareness

**Status:** Accepted — pending implementation
**Date:** 2026-04-18
**Authors:** jaresty

---

## Context

A review of Graphify (a knowledge-graph tool for codebases) and ongoing use of `nn` as an
LLM-driven Zettelkasten surfaced several feature gaps across three areas:

**Graph intelligence** — the link graph is queryable but not yet analysed for structure.
High-connectivity "hub" notes, shortest paths between ideas, and topological clusters are
all computable from existing data but not exposed.

**Note quality** — the atomicity warning on write (ADR-0006, decision 7) flags large notes
at creation time but provides no ongoing visibility. Notes that grow over time via `--append`
can become non-atomic without any feedback. `nn status` reports orphans and broken links but
not bloated notes.

**Link provenance and confidence** — all links currently look identical regardless of whether
they were explicitly created by a human, inferred by an LLM with high confidence, or
speculatively suggested. This makes it hard to triage which relationships to verify.

**Code awareness** — `nn` operates on ideas about code but has no way to read code structure.
An LLM using `nn` must read raw source files to understand them, consuming large amounts of
context. A lightweight structural outline of a code file would reduce that cost significantly.
Graphify demonstrated 71.5x token reduction for mixed corpora using graph-based summaries;
even a simpler per-file outline would yield meaningful savings.

---

## Decisions

### 1. `nn status` long-note report

`nn status` adds a "long notes" section listing notes whose body exceeds `atomicityThreshold`
(currently 2000 chars). Each entry shows ID, title, and character count. The section is
omitted when no long notes exist.

Plain text:
```
long notes (3):
  20260418-0001  Dense Concept Note       4312 chars
  20260418-0002  Sprawling Model          3100 chars
```

JSON: adds `long_notes` array with `id`, `title`, `body_len` fields.

### 2. `nn list --long`

Filter to notes exceeding `atomicityThreshold`. Composes with existing filters (`--type`,
`--search`, `--sort`, `--json`). Reuses the same constant as the write-time warning and
`nn status` report for consistency.

```
nn list --long
nn list --long --sort modified --json
```

### 3. God nodes in `nn status`

`nn status` adds a "hub notes" section listing the top N notes by total link degree
(inbound + outbound). Helps identify the conceptual anchors of the notebook. N=5 by default,
configurable with `--hubs N`. Omitted from status when the notebook has fewer than 10 notes
(too sparse to be meaningful).

```
hub notes (top 5 by link degree):
  20260418-0001  BM25 Search              degree 12
  20260418-0002  Protocol Type            degree 9
```

JSON: adds `hub_notes` array with `id`, `title`, `degree` fields.

### 4. `nn path <id-a> <id-b>`

Find and print the shortest undirected path between two notes via their link graph. Output
is a sequence of note IDs and titles. Returns an error when no path exists.

```
nn path <id-a> <id-b>
nn path <id-a> <id-b> --json
```

Implemented as BFS over the in-memory link graph (already constructed at list time). No
index changes required.

### 5. Link status

Links gain an optional `status` field mirroring the note review workflow: `draft` (default)
or `reviewed`. A `draft` link was created but not yet endorsed by a human; `reviewed` means
a human has examined and accepted the relationship.

`nn link` and `nn bulk-link` default to `draft`. Pass `--status reviewed` to mark as
endorsed at creation time (e.g. when a human is explicitly creating the link).

`nn update-link` and `nn bulk-update-link` accept `--status reviewed` to sign off on
existing links, individually or in bulk.

`nn links <id>` shows link status when present. `nn links <id> --status draft` filters to
unreviewed links for triage.

`nn status` reports the count of draft (unreviewed) links alongside orphans and long notes.

Existing links without a status field are treated as `reviewed` for backward compatibility —
they predate this feature and were created deliberately.

LLMs should express uncertainty via the annotation text ("possibly extends — needs
verification") rather than a separate field. Status answers the binary question a human cares
about: has this relationship been endorsed or not?

### 7. `nn ast <file>`

Print a compact structural outline of a source file suitable for LLM consumption. Uses
`gotreesitter` (pure Go, no CGo, 206 grammars, MIT licensed) to parse the file and emit a
language-appropriate outline: imports, types/classes/structs, function/method signatures,
constants.

The output is designed to replace reading the raw file when the LLM only needs to understand
structure, not implementation.

```
nn ast src/backend/gitlocal.go
nn ast --json src/backend/gitlocal.go
nn ast src/backend/gitlocal.go --trace Write
nn ast src/backend/gitlocal.go --trace Write --root ./
```

Text output (compact, one entry per line):
```
file: src/backend/gitlocal.go  language: go
imports: fmt, os, path/filepath, ...
type Backend struct
  func (b *Backend) Write(n *note.Note) error
  func (b *Backend) Read(id string) (*note.Note, error)
  func (b *Backend) List() ([]*note.Note, error)
  ...
```

JSON output: structured array of symbols with `kind`, `name`, `signature`, `line`.

**`--trace <name>`** appends a name-based reference search across the codebase rooted at
`--root` (default: current directory). Output lists every file and line where the name
appears as a plain-text match. This is explicitly not symbol-aware — the same name in
different scopes or packages will both appear. A disclaimer is printed in the output:

```
references to "Write" (name-match only — not symbol-resolved, may include false positives):
  internal/backend/backend.go:27    Write(n *note.Note) error
  cmd/nn/cmd/new.go:45              state.backend.Write(n)
  cmd/nn/cmd/update.go:65           state.backend.Write(n)
```

The LLM is expected to filter false positives using context. A future decision may add
cross-file symbol resolution via a code index, but that is explicitly deferred — maintaining
a symbol index is a meaningful scope expansion and is not part of this decision.

`gotreesitter` is the first third-party dependency in `nn`. It is isolated to an `internal/ast`
package so it can be replaced or removed without affecting the rest of the codebase. The
pure-Go implementation preserves single-binary distribution via `go install`.

### 8. `nn new --from-file <path>` and `nn new --from-stdin`

Scaffold a new note pre-populated with structured content:

- `--from-file <path>`: runs `nn ast` on the file and uses the outline as the note body scaffold.
  The LLM (or human) fills in the actual insight. Sets title to the filename by default.
- `--from-stdin`: reads text from stdin and uses it as the note body.

These reduce the friction of creating notes about code or external content without copy-pasting.

---

## Implementation Order (ease → complexity)

1. `nn list --long` ☐
2. `nn status` long notes ☐
3. `nn status` hub notes ☐
4. `nn path` ☐
5. `nn new --from-stdin` ☐
6. Link status (`draft` / `reviewed`) ☐
7. `nn ast` (gotreesitter) ☐
8. `nn ast --trace` (name-match reference search) ☐
9. `nn new --from-file` (depends on `nn ast`) ☐

---

## Alternatives Considered

**Leiden community detection (`nn clusters`):** Computable from the link graph but requires
a graph clustering algorithm implementation. Deferred — the hub-notes feature in `nn status`
provides the most actionable signal (identify anchors) without the complexity of full
community detection. Can be revisited when the graph is denser.

**Confidence as a review status enum instead of float:** Simpler schema (`reviewed` |
`unreviewed` | `rejected`) but loses the gradient. Float preserves LLM-native output
(models naturally produce probability scores) and allows threshold-based filtering.

**Embedding-based similarity search:** Rejected again (as in ADR-0006) — heavy dependencies
incompatible with nn's philosophy. BM25 remains the search strategy.

**Link provenance flag + confidence score:** Rejected in favour of link status. Provenance
(`human` / `inferred`) is mostly noise — in practice the LLM creates nearly all links.
Confidence as a float (0.0–1.0) adds schema complexity and requires threshold decisions.
Both are superseded by the simpler binary question: has a human reviewed this link or not?
Uncertainty is better expressed in the annotation text than in a separate field.

**tree-sitter via CGo bindings (`smacker/go-tree-sitter`):** Rejected in favour of
`gotreesitter` (pure Go, no CGo). Single-binary distribution must be preserved.

**Cross-file symbol resolution for `--trace`:** Deferred. Resolving `Write` in `a.go` to
the specific `Write` defined in `b.go` requires a cross-file symbol index — a persistent
data structure that maps names to definitions across the whole codebase. This is a meaningful
scope expansion (analogous to a language server's workspace index) and is explicitly not part
of this ADR. The name-match approach in `--trace` is honest about its limitations and covers
the common case where the LLM needs candidate references, not a verified call graph.
