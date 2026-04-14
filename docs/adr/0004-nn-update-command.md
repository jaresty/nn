# ADR-0004: nn update Command

**Status:** Accepted — implemented
**Date:** 2026-04-14
**Authors:** jaresty

**Implementation log:**
- 2026-04-14 Implemented: `nn update <id>` with --title, --tags, --content, --append, --no-edit flags

---

## Context

There is currently no way to modify a note without either direct file editing or
delete-and-recreate. Delete-and-recreate loses the note ID and breaks all incoming links.
Direct file editing is safe (the index is a cache) but bypasses `nn` and produces no git commit.

The `nn new --no-edit --content` pattern for creation is the established non-interactive
contract. `nn update` should extend that contract to modification.

---

## Decisions

### Command: `nn update <id>`

```
nn update <id> [--title TEXT] [--tags TEXT] [--content TEXT] [--append TEXT] [--no-edit]
```

| Flag | Effect |
|---|---|
| `--title TEXT` | Replace note title |
| `--tags TEXT` | Replace tags (comma-separated) |
| `--content TEXT` | Replace note body entirely |
| `--append TEXT` | Append text to note body (newline-separated) |
| `--no-edit` | Skip `$EDITOR` launch (always use in non-TTY/LLM context) |

At least one flag is required. `--content` and `--append` are mutually exclusive.

The command sets `modified` to the current time and produces one git commit:
`note: update <id> — <title>`

### Direct file editing safety

Direct Markdown edits are safe as long as frontmatter schema is preserved. The SQLite index
is purely a cache; `nn index` rebuilds it completely from files. No hidden state exists
outside the Markdown files and git history.

---

## Consequences

- Notes can be corrected without losing their ID or breaking incoming links
- The LLM can update note content after creation without re-linking
- `--append` supports incremental refinement (add to a note over time)
