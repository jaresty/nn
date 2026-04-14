# ADR-0003: List Sort, Bulk Link, FTS Ranking, and Link Types

**Status:** Accepted — implemented
**Date:** 2026-04-14
**Authors:** jaresty

**Implementation log:**
- 2026-04-14 All four items implemented: `nn list --sort`, `nn bulk-link`, ranked `--search`, typed links with `--type` flag

---

## Context

Four feature requests arrived from real-world use with 50+ notes:

1. **Sort order on `nn list`** — results are returned in filesystem order; recently-modified
   notes are hard to find for linking.
2. **Bulk linking** — creating N links from one note requires N sequential `nn link` commands,
   each producing its own git commit. At 4–5 links per session this is noisy.
3. **Ranked full-text search** — `--search` is a case-insensitive substring filter, not a
   ranked search. With 50+ notes, the right link target is often buried.
4. **Typed links** — links carry annotations (text) but no machine-readable relationship type.
   Users want to express `refines`, `contradicts`, `source-of`, etc. as queryable metadata.

---

## Decisions

### 1. `nn list --sort <field>`

Add a `--sort` flag to `nn list` accepting `title`, `created`, `modified` (default: `created`
descending, matching current behaviour which returns notes in ID order — IDs are timestamps).

Sort is applied after filtering, before `--limit`.

`--sort modified` sorts descending (most-recently-modified first) since that is the
dominant use case. All sort fields are descending.

```
nn list --sort modified
nn list --sort title
nn list --sort created      # default
```

No index change required — sorting is an in-memory operation on the already-loaded note slice.

### 2. `nn bulk-link <from-id> --to <id> --annotation <text> [--to <id> --annotation <text>]...`

A new `nn bulk-link` command accepts one `--to`/`--annotation` pair per target and creates
all links in a single operation. The `gitlocal` backend receives a new `AddLinks` method that
writes all link additions to the note files and produces **one git commit** covering all of them.

Cobra does not natively support repeated `--to`/`--annotation` pairs in positional order.
The implementation uses `--StringArrayVar` for `--to` and `--annotation` separately and
zips them by position — if the counts differ, the command errors.

```
nn bulk-link <from-id> \
  --to <id1> --annotation "extends this" \
  --to <id2> --annotation "contradicts that"
```

Single commit message: `note: bulk-link <from-id> → <N> notes`

### 3. `nn list --search` ranked FTS via SQLite FTS5

Replace the in-memory substring filter with SQLite FTS5 when `--search` is used.

**Schema addition** (additive, `nn index` rebuild required):

```sql
CREATE VIRTUAL TABLE notes_fts USING fts5(
  id UNINDEXED,
  title,
  body,
  annotations,   -- concatenated link annotations for this note
  content=notes,
  tokenize='unicode61'
);
```

**Ranking**: `bm25(notes_fts, 0, 10, 1, 2)` — weights title (10×) > annotations (2×) > body
(1×). The `id` column is unindexed. Results are ordered by rank descending.

**Compatibility**: when `--search` is used with other filters, FTS runs first (returning
ranked IDs), then the existing filter pipeline applies to the matched notes. The in-memory
`containsFold` path is removed.

`nn index` populates `notes_fts` on rebuild. Incremental updates happen in `AddLink`,
`RemoveLink`, `Write`, and `Delete` backend operations.

### 4. Typed links

Add an optional `type` field to the `Link` struct and the Markdown link format.

**Markdown format** (backwards-compatible):

```markdown
## Links

- [[20260411090000-1234]] — provides the foundational philosophy  (existing format, no type)
- [[20260411090000-1234]] [refines] — extends the definition
```

The type is bracketed, placed between the `]]` and the `—` separator. Parsing is additive:
existing links without a type remain valid; `Type` is empty string when absent.

**Link type vocabulary** (open, not enforced by the CLI):
`refines`, `contradicts`, `source-of`, `extends`, `supports`, `questions`

The CLI does not validate link types — any non-empty string is accepted. This mirrors the
`tags` design (open vocabulary). The `--type` flag on `nn link` and `nn bulk-link` accepts
any string.

**Schema addition** to the `links` index table:

```sql
ALTER TABLE links ADD COLUMN type TEXT NOT NULL DEFAULT '';
```

`nn link --type refines`, `nn links <id> [--type <filter>]`, and `nn graph --json` all
surface the type field.

---

## Implementation order

1. `nn list --sort` — in-memory sort, no schema change (tiny)
2. `nn bulk-link` — new command + `AddLinks` backend method (small)
3. FTS ranked search — index schema + backend wiring (medium)
4. Typed links — schema change + parse/marshal update (medium-large)

---

## Consequences

- `--sort modified` makes recently-touched notes immediately discoverable
- `bulk-link` reduces link-creation noise from N commits to 1
- FTS ranking makes search useful at 50+ notes without external tooling
- Typed links make the graph semantically richer and queryable by relationship kind
- Link type vocabulary is intentionally open — no ontology lock-in

---

## Alternatives Considered

**`--sort` ascending/descending flag:** Omitted for simplicity — descending is the right
default for all three fields in practice. Can be added later with `--asc`.

**`nn bulk-link` with JSON input:** A `--from-json` flag accepting `[{"to":"...","annotation":"..."}]`
was considered. Rejected for M1 — the repeated-flag form is shell-friendly and composable.
JSON input can be added later.

**External FTS (Bleve, Tantivy via CGo):** Rejected. SQLite FTS5 is already present,
requires no new dependency, and is sufficient for notebooks up to tens of thousands of notes.

**Enforced link type vocabulary:** Rejected. Enforcing a closed set makes the schema a
dependency. The open vocabulary matches the `tags` design principle.
