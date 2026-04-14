# nn-guide

Reference for `nn` commands, flags, and LLM usage patterns.

## Global flags (all commands)

```
--json          Machine-readable JSON output
--no-color      Disable ANSI color
-q, --quiet     Suppress progress/info output
--notebook      Select a non-default notebook (name from config)
```

## nn new

Create a new note.

```
nn new --title TEXT --type TYPE [--tags TEXT] [--content TEXT] [--no-edit]
       [--link-to ID --annotation TEXT]
```

- `--type` is required: `concept | argument | model | hypothesis | observation`
- `--no-edit` skips `$EDITOR` launch (always use in non-TTY/LLM context)
- `--content TEXT` sets the note body directly

### Choosing a type

The five types cover the epistemic roles a note can play (after Ahrens, *How to Take Smart Notes*):

| Type | Use when the note… | Example title |
|---|---|---|
| `concept` | defines or explains a single idea, term, or principle | "The Atomicity Principle" |
| `argument` | makes a claim and supports it with reasoning | "Atomicity enables reuse across contexts" |
| `model` | describes a framework, pattern, or mental model | "The Zettelkasten as a second brain" |
| `hypothesis` | states an untested conjecture worth investigating | "Dense linking predicts note longevity" |
| `observation` | records a concrete fact, datum, or empirical finding | "Luhmann produced 90,000 notes over 40 years" |
| `question` | poses an open question that the graph should eventually answer | "Why did Luhmann avoid hierarchical folders?" |

**Decision heuristic:** if you're not sure, ask — *is this a definition (concept), a claim with support (argument), a framework (model), a guess to test (hypothesis), or something I witnessed/measured (observation)?* If none fit cleanly, the note may not be atomic yet.

## nn show

Print note content to stdout. Accepts a full ID or a title substring.

```
nn show <id-or-title>
```

If the query doesn't match an ID exactly, `nn` searches note titles case-insensitively.
If multiple titles match, the command lists the candidates and exits with an error — use
the full ID to disambiguate.

## nn list

List and filter notes.

```
nn list [--tag TEXT] [--type TYPE] [--status STATUS]
        [--linked-from ID] [--linked-to ID] [--orphan]
        [--search TEXT] [--sort FIELD] [--limit N] [--json]
```

`--search TEXT` performs a ranked case-insensitive search across note title and body. Title matches rank above body matches.

`--sort FIELD` sorts results: `title` (alphabetical), `modified` (most-recently-modified first), `created` (default, most-recently-created first). Applied after filtering and ranking.

Filters compose: `nn list --search "implicit" --type concept --sort modified` works as expected.

## nn update-link

```
nn update-link <from-id> <to-id> [--annotation TEXT] [--type TYPE]
```

Updates the annotation and/or type of an existing link in place — no unlink/relink needed. At least one flag is required. Only the specified fields are changed; unspecified fields are preserved.

## nn link / nn unlink / nn bulk-link

```
nn link <from-id> <to-id> --annotation "relationship description" [--type TYPE]
nn unlink <from-id> <to-id>
nn bulk-link <from-id> --to <id> --annotation "..." [--to <id> --annotation "..."]...
```

Annotations are required. A bare link is a schema violation.

`--type` is optional. Open vocabulary: `refines`, `contradicts`, `source-of`, `extends`, `supports`, `questions`.

`nn bulk-link` creates all links in a single git commit. `--to` and `--annotation` are paired by position; counts must match.

## nn graph

```
nn graph [--json]
```

JSON output: `{ "nodes": [...], "edges": [...] }`

## nn status

```
nn status [--json]
```

Reports: total notes, orphan count (with IDs and titles), draft count, broken links.

`--json` output: `{ "total": N, "orphans": [{"id": "...", "title": "..."}], "drafts": N, "broken_links": [...] }`

## nn links

```
nn links <id> [--type TYPE] [--json]
```

Lists outgoing links from a note with their annotations. `--type` filters to a specific relationship type.

Text output: one entry per link — `targetID  title\n  [type] annotation` (type shown when present)

`--json` output: `[{"id": "...", "title": "...", "annotation": "...", "type": "..."}]` (type omitted when empty)

## nn update

```
nn update <id> [--title TEXT] [--tags TEXT] [--content TEXT] [--append TEXT] [--no-edit]
```

At least one change flag is required. `--content` and `--append` are mutually exclusive.

| Flag | Effect |
|---|---|
| `--title TEXT` | Replace note title |
| `--tags TEXT` | Replace all tags (comma-separated) |
| `--content TEXT` | Replace note body entirely |
| `--append TEXT` | Append text to note body (double-newline separator) |
| `--no-edit` | Skip `$EDITOR` (always use in non-TTY/LLM context) |

Direct file editing is also safe — the index is a cache rebuilt from files.

## nn promote

```
nn promote <id> --to reviewed|permanent
```

Status progression: `draft` → `reviewed` → `permanent`.

## nn delete

```
nn delete <id> --confirm
```

`--confirm` is required. Warns if other notes link to the deleted note.

## nn install-skills

```
nn install-skills [--dest DIR] [--list]
```

Copies skill directories into `~/.claude/skills/` (or `--dest`).

## Configuration

`~/.config/nn/config.toml`:

```toml
[notebooks]
default = "personal"

[notebooks.personal]
path = "~/notes"
backend = "gitlocal"
```

Environment overrides:
- `NN_NOTEBOOK` — select a named notebook (overrides `default`)
- `NN_CONFIG_DIR` — override config directory (useful for testing)

## Note schema

```yaml
---
id: 20260411120045-3821
title: "The Atomicity Principle"
type: concept
status: draft
tags: [zettelkasten, methodology]
created: 2026-04-11T12:00:45Z
modified: 2026-04-11T12:05:00Z
---

Body text.

## Links

- [[20260411090000-1234]] — provides the foundational philosophy this principle implements
```

## LLM usage patterns

**Create and link in sequence:**
```
nn new --title "Concept A" --type concept --content "..." --no-edit
# note ID from output: 20260411120045-0001
nn list --json | jq '.[].id'   # find related note IDs
nn link 20260411120045-0001 <related-id> --annotation "extends this concept"
```

**Find orphans before a review session:**
```
nn list --orphan --json
```

**Export graph for visualisation:**
```
nn graph --json > graph.json
```
