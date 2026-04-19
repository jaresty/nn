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
       [--from-stdin]
       [--from-file PATH]
```

- `--type` is required: `concept | argument | model | hypothesis | observation | question | protocol`
- `--no-edit` skips `$EDITOR` launch (always use in non-TTY/LLM context)
- `--content TEXT` sets the note body directly
- `--from-stdin` reads the note body from stdin
- `--from-file PATH` scaffolds the note body from `nn ast` output for a source file (sets title to filename if not given)

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
| `protocol` | specifies an imperative procedure the LLM should follow in this notebook | "When creating a hypothesis, immediately link it to its source observation" |

**Decision heuristic:** if you're not sure, ask — *is this a definition (concept), a claim with support (argument), a framework (model), a guess to test (hypothesis), something I witnessed/measured (observation), an open question (question), or an operating instruction for the LLM (protocol)?* If none fit cleanly, the note may not be atomic yet.

## nn show

Print note content to stdout. Accepts a full ID or a title substring.

```
nn show <id-or-title> [--depth N] [--json]
nn show --linked-from <id>
```

If the query doesn't match an ID exactly, `nn` searches note titles case-insensitively.
If multiple titles match, the command lists the candidates and exits with an error — use
the full ID to disambiguate.

`--depth N` traverses outgoing links from the given note up to N hops, collecting all
reachable notes and printing them as a single concatenated Markdown document separated by
`---`. Useful for loading a coherent subgraph as context for an LLM.

```
nn show <id> --depth 2                 # root + 2 hops of outgoing links
nn show <id> --depth 1 --json          # JSON array with depth field per note
```

`--json` with `--depth` returns an array of note objects in BFS order, each with an added
`depth` field (0 = origin note, 1 = direct links, etc.).

## nn list

List and filter notes.

```
nn list [--tag TEXT] [--type TYPE] [--status STATUS]
        [--linked-from ID] [--linked-to ID] [--orphan] [--global] [--long]
        [--search TEXT] [--similar ID] [--sort FIELD] [--limit N] [--json]
```

`--search TEXT` performs a ranked case-insensitive search across note title and body. Title matches rank above body matches.

`--similar ID` ranks all notes by BM25 similarity to the given note's title and body, excluding the note itself. Use for serendipitous discovery — find notes that share vocabulary with a given note but have no explicit link. Composes with `--status`, `--tag`, `--type`, `--limit`, `--json`. When `--similar` is active, `--sort` is ignored (similarity ranking takes precedence).

```
nn list --similar <id>                 # notes most similar to <id>
nn list --similar <id> --limit 5       # top 5 most similar
nn list --similar <id> --status permanent --json
```

`--sort FIELD` sorts results: `title` (alphabetical), `modified` (most-recently-modified first), `created` (default, most-recently-created first). Applied after filtering and ranking. Ignored when `--similar` is active.

`--global` returns protocol notes with no outgoing `governs` links — protocols that apply universally to the entire notebook rather than governing specific notes. Distinct from `--orphan`: a global protocol is intentionally universal, not forgotten.

`--long` filters to notes whose body exceeds the atomicity threshold (2000 chars). Use to find notes that have grown too large to split.

Filters compose: `nn list --search "implicit" --type concept --sort modified` works as expected.

## nn update-link / nn bulk-update-link

```
nn update-link <from-id> <to-id> [--annotation TEXT] [--type TYPE] [--status draft|reviewed]
nn bulk-update-link <from-id> --to <id> [--type TYPE] [--annotation TEXT] [--status draft|reviewed] [--to <id> ...]
```

Update annotation, type, and/or status of existing links in place — no unlink/relink needed. At least one change flag is required. Only specified fields are modified; unspecified fields are preserved.

`--status reviewed` signs off a draft link as human-endorsed. Use after verifying LLM-suggested links.

`nn bulk-update-link` applies all updates in a single git commit. `--type` and `--annotation` are paired with `--to` by position; if provided, their counts must match `--to`. `--status` applies to all `--to` targets.

## nn link / nn unlink / nn bulk-link

```
nn link <from-id> <to-id> --annotation "relationship description" --type TYPE [--status draft|reviewed]
nn unlink <from-id> <to-id>
nn bulk-link <from-id> --to <id> --annotation "..." --type TYPE [--status draft|reviewed] [--to <id> --annotation "..." --type TYPE]...
```

Both `--annotation` and `--type` are required. A bare link is a schema violation.

Canonical types: `refines`, `contradicts`, `source-of`, `extends`, `supports`, `questions`, `governs`.

`--status` defaults to `draft`. Pass `--status reviewed` when a human is explicitly creating and endorsing the link at creation time.

`nn bulk-link` creates all links in a single git commit. `--to`, `--annotation`, and `--type` are paired by position; counts must match. `--status` applies to all targets.

## nn graph

```
nn graph [--json]
```

JSON output: `{ "nodes": [...], "edges": [...] }`

## nn status

```
nn status [--json] [--hubs N]
```

Reports: total notes, orphan count (with IDs/titles), draft count, broken links, draft link count, long notes, hub notes.

- **long notes**: notes whose body exceeds 2000 chars — candidates for splitting. Section omitted when none exist.
- **hub notes**: top N notes by link degree (inbound + outbound). Only shown when notebook has ≥10 notes. `--hubs N` overrides the default of 5.
- **draft links**: count of links with `status: draft` — links not yet human-endorsed.

`--json` output adds: `"draft_links": N`, `"long_notes": [{"id": "...", "title": "...", "body_len": N}]`, `"hub_notes": [{"id": "...", "title": "...", "degree": N}]`

## nn links

```
nn links <id> [--type TYPE] [--status draft|reviewed] [--json]
```

Lists outgoing links from a note with their annotations. `--type` filters by relationship type. `--status` filters by link status.

Link status: `draft` (default for new links — not yet human-endorsed), `reviewed` (human has verified the relationship). Legacy links without a status field are treated as `reviewed` for backward compatibility.

Text output: one entry per link — `targetID  title {status}\n  [type] annotation` (status and type shown when present)

`--json` output: `[{"id": "...", "title": "...", "annotation": "...", "type": "...", "status": "..."}]`

**Triage draft links:** `nn links <id> --status draft` shows only unreviewed links for a specific note.

## nn path

```
nn path <id-a> <id-b> [--json]
```

Find and print the shortest undirected path between two notes via the link graph (BFS). Returns an error when no path exists.

Text output: each note on its own line with an `→` separator between hops.

`--json` output: `[{"id": "...", "title": "..."}]` — ordered path from A to B.

## nn clusters

```
nn clusters [--min N] [--singletons] [--json]
```

Detect topological clusters of notes using label propagation. Each note starts with its own label and iteratively adopts the most common label among its linked neighbours.

- `--min N` omits clusters smaller than N notes (default: 2). Notes with no links are singletons and omitted by default.
- `--singletons` includes singleton clusters (notes with no links).

Text output: one cluster per block — `cluster N (K notes):\n  ID  Title\n  ...`

`--json` output: `[{"notes": [{"id": "...", "title": "..."}]}]`

## nn ast

```
nn ast <file> [--json] [--trace] [--root DIR]
```

Print a compact structural outline of a source file (imports, types, functions, constants). Uses gotreesitter (pure Go) to parse the file.

Supported languages: Go, Python, JavaScript, TypeScript, Rust, Java.

Text output:
```
file: src/backend/gitlocal.go  language: go
imports: fmt, os, path/filepath, ...
type Backend struct {
func (b *Backend) Write(n *note.Note) error {
...
```

`--json` output: `[{"kind": "...", "name": "...", "signature": "...", "line": N}]`

`--trace` searches for name-match references to every symbol in the outline across the codebase rooted at `--root` (default: `.`). Emits one `references to "X"` section per symbol. Explicitly name-match only — not symbol-resolved, may include false positives.

```
nn ast src/backend/gitlocal.go --trace --root ./
```

## nn update

```
nn update <id> [--title TEXT] [--tags TEXT] [--content TEXT] [--append TEXT] [--type TYPE] [--no-edit]
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

## nn random

Return a randomly selected note. Optionally filtered.

```
nn random [--tag TEXT] [--type TYPE] [--status STATUS] [--json]
```

Returns one note at random from the notebook. All filters from `nn list` are supported.
Use for deliberate serendipity — re-encounter a forgotten note and consider whether it
connects to current work.

```
nn random                         # any note
nn random --status permanent      # a random permanent note
nn random --tag philosophy --json
```

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

- [[20260411090000-1234]] [extends] {draft} — provides the foundational philosophy this principle implements
```

Link format: `- [[target-id]] [type] {status} — annotation`
- `[type]` optional: `refines`, `contradicts`, `source-of`, `extends`, `supports`, `questions`, `governs`
- `{status}` optional: `draft` (default for new links), `reviewed` (human-endorsed). Absent = `reviewed` (legacy compat).

## LLM usage patterns

**Create and link in sequence:**
```
nn new --title "Concept A" --type concept --content "..." --no-edit
# note ID from output: 20260411120045-0001
nn list --json | jq '.[].id'   # find related note IDs
nn link 20260411120045-0001 <related-id> --annotation "extends this concept" --type extends
```

**Find orphans before a review session:**
```
nn list --orphan --json
```

**Export graph for visualisation:**
```
nn graph --json > graph.json
```

**Discover related notes (no known query):**
```
nn list --similar <id> --limit 10
```

**Load a topic cluster as LLM context:**
```
nn show <id> --depth 2
```

**Serendipitous re-encounter:**
```
nn random --status permanent
```
