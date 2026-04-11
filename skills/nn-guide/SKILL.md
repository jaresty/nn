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
        [--search TEXT] [--limit N] [--json]
```

`--search TEXT` performs a case-insensitive substring match across note title and body.
Filters compose: `nn list --search "implicit" --type concept` works as expected.

## nn link / nn unlink

```
nn link <from-id> <to-id> --annotation "relationship description"
nn unlink <from-id> <to-id>
```

Annotations are required. A bare link is a schema violation.

## nn graph

```
nn graph [--json]
```

JSON output: `{ "nodes": [...], "edges": [...] }`

## nn status

```
nn status
```

Reports: total notes, orphan count, draft count, broken links.

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
