# ADR-0006: LLM Agent Ergonomics

**Status:** Accepted — pending implementation
**Date:** 2026-04-16
**Authors:** jaresty

---

## Context

A friction analysis of `nn` from an LLM-agent perspective identified six root causes of
unnecessary tool-call round-trips and poor discoverability:

1. No batch read — assembling context requires O(n) `nn show` calls
2. `nn list --json` metadata too sparse to triage without reading each note
3. No temporal filtering — can't ask "what changed since last session"
4. No bulk note creation — building a cluster requires N `nn new` + M `nn link` calls
5. Search is substring-only — LLMs must guess exact keywords; multi-word queries degrade
6. Link type vocabulary drifts — open strings make relationship semantics unreliable

A seventh issue was raised: notes with large bodies may violate the atomicity principle
without any feedback at write time.

Semantic search via an external tool (qmd, embedding models) was considered and rejected —
heavy dependencies (node-llama-cpp, GGUF models) are incompatible with nn's philosophy of
a lightweight, plain-text, pure-Go CLI.

---

## Decisions

### 1. Multi-ID `nn show`

`nn show` accepts one or more IDs or title substrings, concatenating output separated by
a `---` marker. Reduces context-building from N tool calls to 1.

```
nn show <id1> <id2> <id3>
```

### 2. Rich `nn list --json`

Add opt-in fields to `nn list --json` output: `created`, `modified`, `link_count`
(outgoing), `body_preview` (first 200 chars). Enabled via `--fields` flag or `--rich`
shorthand.

```
nn list --json --rich
nn list --json --fields id,title,type,modified,link_count,body_preview
```

Eliminates the "read to triage" loop — an LLM can sort by `modified`, filter by
`link_count > 0`, and scan `body_preview` without individual reads.

### 3. `nn list --since`

Add `--since <datetime>` and `--before <datetime>` flags to `nn list`. Accepts ISO 8601
dates (`2026-04-15`) or datetimes (`2026-04-15T10:00:00Z`). Filters on `modified` time.

```
nn list --since "2026-04-15" --json
```

### 4. `nn bulk-new`

Accept a JSON array of note specs on stdin or via `--json`, creating all notes and their
declared links in a single git commit. Links within the batch reference other notes by
their position index (`{"ref": 0}`).

```
nn bulk-new --json '[
  {"title": "Note A", "type": "concept", "content": "..."},
  {"title": "Note B", "type": "argument", "content": "...",
   "links": [{"ref": 0, "annotation": "extends"}]}
]'
```

### 5. BM25 search

Replace the current title×10/body×1 scoring in `nn search` with BM25 (Best Match 25),
implemented in pure Go with no external dependencies. BM25 handles multi-word queries,
term frequency saturation, and document length normalization — significantly better than
substring scoring for the queries LLMs actually make.

The existing `nn search` command is updated in place; `nn list --search` retains its
alias relationship.

### 6. Link type allow list

Add a hardcoded canonical set of link types to `internal/note`:

```
refines | contradicts | source-of | extends | supports | questions | governs
```

(`governs` added for protocol→note relationships per ADR-0005.)

`nn link --type <value>` warns (stderr, exit 0) when the type is not in the allow list.
`nn status` reports unknown-type link count. The allow list is enforced as a warning,
not a hard error, to preserve flexibility for experimental types.

### 7. Atomicity warning on large notes

`nn new` and `nn update` warn (stderr, exit 0) when the note body exceeds 2000 characters.
Warning text: `warning: note body is N chars — consider splitting into atomic notes`.
The threshold is a constant in `internal/note`; no config required.

---

## Implementation Order (ease → complexity)

1. `nn list --since` — flag + filter, ~20 lines
2. Atomicity size warning — threshold check, ~10 lines
3. Multi-ID `nn show` — variadic args, ~15 lines
4. Rich `nn list --json` — struct extension + `--rich`/`--fields` flag
5. Link type allow list — constant set + warn path
6. `nn bulk-new` — JSON parsing + batch write + single commit
7. BM25 search — scoring algorithm replacement, pure Go

---

## Alternatives Considered

**Semantic search via qmd / embedding models:** Rejected. Heavy dependencies
(node-llama-cpp, GGUF model files) are incompatible with nn's lightweight philosophy.
BM25 in pure Go provides meaningful improvement without any new dependencies.

**Link type enforcement as hard error:** Rejected. Experimental types are valuable
during note-taking sessions. Warning preserves flexibility while surfacing drift.

**`--fields` selector syntax for list:** May be over-engineered for now. `--rich`
shorthand enables the common case; `--fields` can be deferred.

**goldmark AST query language:** Interesting but large scope. Deferred — the improvements
above address the most acute friction points without requiring a query language.
