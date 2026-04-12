# ADR-0001: nn — LLM-Driven Zettelkasten CLI Architecture

**Status:** Accepted — M0–M6 implemented
**Date:** 2026-04-11
**Authors:** jaresty

**Implementation log:**
- 2026-04-11 M0: repo bootstrap, CLAUDE.md, ADR, `.gitignore`, `go.mod`
- 2026-04-11 M1: `internal/note`, `internal/backend/gitlocal`, `internal/index` — all tested
- 2026-04-11 M2–M6: full CLI (`nn new`, `nn show`, `nn list`, `nn link`, `nn unlink`, `nn graph`, `nn status`, `nn promote`, `nn delete`, `nn install-skills`) — all tested
- 2026-04-11 M7: GoReleaser + GitHub Actions CI (`test.yml`, `release.yml`) + Homebrew tap (`jaresty/homebrew-nn`); MCP server ruled out (skills cover LLM integration)
- 2026-04-12 `question` added as sixth note type; `reference` deferred

---

## Context

The goal is a CLI tool (`nn`) that lets an LLM organize a Zettelkasten note graph on behalf of
a user. The LLM drives the CLI as a subprocess via shell commands. The system must also be usable
interactively by humans.

A Zettelkasten is a personal knowledge system where:
- Notes are **atomic** — each contains exactly one idea (concept, argument, model, hypothesis, or observation)
- Notes are **linked** — every link carries an explicit annotation explaining the relationship
- The note **graph** is the primary artifact; folder hierarchy is explicitly avoided
- Notes are written in the author's own words (not direct quotation)
- The system grows organically through ongoing linking, not upfront classification

Prior art surveyed: `zk-org/zk` (Go, feature-complete), `sirupsen/zk` (shell/Ruby, minimal),
Obsidian, The Archive, Logseq, Org-roam. The name `nn` was chosen because `zk` is already taken
by both of the above tools.

---

## Decisions

### 1. Name: `nn`

`nn` is short, typable, and available. It reads as "notes" without being on the nose. The binary
is named `nn`; the module is `github.com/jaresty/nn`.

### 2. Language: Go

Single statically-linked binary with no runtime dependencies. Matches `zk-org/zk` precedent.
Fast startup (critical for LLM tool-call latency). Good library ecosystem for CLI (cobra),
Markdown parsing, SQLite, and YAML frontmatter.

### 3. Global Configuration — Notebook Not Tied to CWD

The notebook location is configured globally, not derived from the current working directory.
This allows `nn` to be called from any directory — from a shell alias, a Raycast script, or
an LLM tool call — without requiring a `cd` first.

Config precedence (highest to lowest):
1. `--notebook <name>` flag (selects a named notebook from config)
2. `$NN_NOTEBOOK` environment variable (overrides default notebook name)
3. `~/.config/nn/config.toml` user config (defines notebooks and default)

Config schema supports multiple named notebooks from the start:

```toml
[notebooks]
default = "personal"

[notebooks.personal]
path = "~/notes"
backend = "gitlocal"

[notebooks.work]
path = "~/work/notes"
backend = "gitlocal"
```

This is a deliberate departure from `zk-org/zk`'s `.zk/` directory marker model. The `.zk/`-
style per-notebook marker is not required, though a notebook directory may contain a
`.nn/` marker for documentation purposes.

### 4. Data Model

**File naming:** `<id>-<slug>.md` where `<id>` is a 14-digit timestamp + 4-digit random suffix:
`20250411120045-3821`. The random suffix prevents collisions when the LLM creates multiple notes
in rapid succession.

**Frontmatter (YAML):**

```yaml
---
id: 20250411120045-3821
title: "The Atomicity Principle"
type: concept          # concept | argument | model | hypothesis | observation
status: draft          # draft | reviewed | permanent
tags: [zettelkasten, methodology]
created: 2025-04-11T12:00:45Z
modified: 2025-04-11T12:05:00Z
---
```

The `id` field in frontmatter is canonical. The filename includes the id for navigability but
the frontmatter value governs. The `type` field is required on creation — it forces the LLM
to make an atomicity judgment before committing the note. The `status` field enables a
review workflow: the LLM creates `draft` notes; humans promote them to `reviewed` or `permanent`.

The `type` vocabulary (`concept`, `argument`, `model`, `hypothesis`, `observation`) is a
synthesis drawn from Sönke Ahrens, *How to Take Smart Notes* (2017), which distinguishes
note types by their epistemic role, and from Niklas Luhmann's original Zettelkasten practice
as documented in Schmidt, J. F. K., "Niklas Luhmann's Card Index" (2016). The specific
five-value taxonomy is an original choice for `nn`; no single source uses this exact set.

The five types are not intended to be exhaustive. Their purpose is not classification but
commitment: requiring a type forces the author (or LLM) to decide what kind of claim the
note makes before writing it, which enforces atomicity. Fewer, sharper types serve this
better than a complete ontology.

One known gap remains at the literature-review end of the workflow:
- `reference` — a summary of an external source (Ahrens' "literature note"; currently
  unrepresented)

`question` was added as a sixth type (see implementation log). `reference` is deferred
deliberately; adding it would be appropriate if `nn` is used heavily for literature review.

**Link section:** Links live in a dedicated `## Links` section at the bottom of the note body:

```markdown
## Links

- [[20250411090000-1234]] — provides the foundational philosophy this principle implements
- [[20250411110000-5678]] — contradicts: argues atomicity is context-dependent
```

Annotations are required. A bare link with no annotation is a schema violation.

### 5. CLI Command Design

**Subcommand structure:** `nn <verb> [flags]`

| Command | Purpose |
|---|---|
| `nn new` | Create a new note |
| `nn show <id>` | Print note content to stdout |
| `nn list` | List/filter notes |
| `nn link <from> <to>` | Add an annotated link between two notes |
| `nn unlink <from> <to>` | Remove a link |
| `nn tags` | List all tags with counts |
| `nn graph` | Output link relationships as JSON or Graphviz dot |
| `nn index` | Rebuild the SQLite index from files |
| `nn status` | Notebook health: orphan notes, draft count, broken links |
| `nn delete <id>` | Delete a note (warns if linked-to by others) |
| `nn promote <id>` | Advance note status: draft → reviewed → permanent |

**Non-interactive contract:** Every value that an interactive prompt would solicit is also
suppliable as a named flag. When stdout is not a TTY, interactive elements are suppressed
automatically. The LLM always uses the flag form.

| Interactive element | Non-interactive equivalent |
|---|---|
| `$EDITOR` launch | `--content TEXT` + `--no-edit` |
| `fzf` selection picker | Filter flags + `--json` output |
| Annotation prompt | `--annotation TEXT` (required in non-TTY mode) |
| Confirmation prompt | `--confirm` flag |
| Status selection | `--to STATUS` |

**Cross-cutting flags (all commands):**

```
--json          Machine-readable JSON output
--no-color      Disable ANSI color (also: NO_COLOR env var)
-q, --quiet     Suppress progress/info output
--notebook      Select a non-default notebook
```

**`nn new` flags:**
```
--title TEXT    Note title
--type TYPE     concept|argument|model|hypothesis|observation (required)
--tags TEXT     Comma-separated tags
--content TEXT  Note body (use with --no-edit)
--no-edit       Skip opening $EDITOR
--link-to ID    Immediately link to an existing note (requires --annotation)
--annotation    Link annotation when using --link-to
```

**`nn list` filters:**
```
--tag TEXT           Filter by tag
--type TYPE          Filter by note type
--status STATUS      Filter by status
--linked-from ID     Notes that link to this ID
--linked-to ID       Notes this ID links to
--orphan             Notes with no links (inbound or outbound)
--created-after DATE
--limit N
--format FORMAT      title|id|path|json (default: title in TTY, id in non-TTY)
```

### 6. Backend Abstraction

A `Backend` interface in `internal/backend/backend.go` decouples note CRUD from storage.
The first implementation is `gitlocal`: notes as `.md` files, index as SQLite in
`~/.config/nn/index.db`, Git commit after each write operation.

The interface is designed so future backends (remote Git, SQLite-primary, S3) can be added
without changing any caller code. The backend is selected per-notebook in config.

**Commit message convention for `gitlocal`:**

```
note: create <id> — <title>
note: link <from-id> → <to-id>
note: unlink <from-id> → <to-id>
note: promote <id> to <status>
note: delete <id> — <title>
note: split <id> into <id1>, <id2>
```

Each `nn` command that modifies state produces exactly one Git commit.

### 7. Index

SQLite database at `~/.config/nn/index.db` (outside the notebook directory so it is not
committed to Git). Rebuilt on demand with `nn index`. Stale detection via `content_hash`
compared to file mtime on every read command. The index is a cache — if deleted, `nn index`
recreates it completely from the Markdown files.

Tables: `notes`, `links`, `tags`.

### 8. Skill Distribution via `nn install-skills`

`nn` ships its own Claude Code skills inside the repository under `skills/`. The command
`nn install-skills` copies them into `~/.claude/skills/` so they become available to Claude
Code in any project.

**Directory layout:**

```
skills/
  nn-workflow/       SKILL.md — multi-step bar-style workflow for nn operations
  nn-guide/          SKILL.md — reference for nn commands, flags, and LLM usage patterns
```

**`nn install-skills` behavior:**
- Copies each `skills/<name>/` directory to `~/.claude/skills/<name>/`
- Overwrites if already present (upgrades on re-run)
- Prints the list of installed skills on success
- `--list` flag prints what would be installed without copying
- `--dry-run` alias for `--list`

Skills are embedded into the binary at build time using Go's `//go:embed skills/` directive,
so `nn install-skills` works without access to the source repository.

**Pattern:** Mirrors how `bar` makes its grammar and skills available — the CLI is
self-contained and installs its own Claude Code integration.

### 9. Implementation Milestones

| Milestone | Scope | Status |
|---|---|---|
| M0 | Repo bootstrap, CLAUDE.md, this ADR, `.gitignore`, `go.mod` | ✅ done |
| M1 | `internal/note` (Note struct, ID gen, frontmatter read/write), `internal/backend/gitlocal` (file ops, git commit wrapper), `internal/index` (SQLite schema, rebuild) — library only, no CLI | ✅ done |
| M2 | `nn new` (with `--no-edit`, `--content` path) and `nn show` — end-to-end create + read | ✅ done |
| M3 | `nn list` with all filters, `--json`, TTY detection | ✅ done |
| M4 | `nn link`, `nn unlink`, `nn graph` | ✅ done |
| M5 | `nn status`, `nn promote`, `nn delete` — review workflow complete | ✅ done |
| M6 | `nn install-skills` command + initial skills (`nn-workflow`, `nn-guide`); `//go:embed` via `skills/embed.go` | ✅ done |
| M7 | Release infrastructure: GoReleaser + GitHub Actions CI/CD + Homebrew tap (`jaresty/homebrew-nn`) | ✅ done |
| M8 | Second backend (remote-git) to validate interface abstraction | pending |
| M9 | `nn list --search TEXT` (full-text on title+body, in-memory); `nn show` title-prefix fallback | ✅ done |

**Implementation notes (deviations from original spec):**

- **ID suffix: counter not random.** ADR specified a 4-digit random suffix; implementation uses a per-second monotonic counter (mutex-protected) to guarantee uniqueness under concurrent generation. Intent preserved: collision prevention.
- **`skills/embed.go` shim.** Go's `//go:embed` does not allow `../` paths, so the embed directive lives in `skills/embed.go` (a `package skills` shim) rather than in the CLI package directly. `nn install-skills` imports `github.com/jaresty/nn/skills`.
- **`NN_CONFIG_DIR` env var.** Added for testability: overrides the config directory without touching `$HOME`. Not in the original spec.
- **`backend.Backend` interface.** Defined in `internal/backend/backend.go` with methods: `Write`, `Read`, `Delete`, `List`, `AddLink`, `RemoveLink`, `Promote`. All commands go through this interface.
- **`.gitignore` scoping.** Changed `nn` to `/nn` to match only the root-level binary, not the `cmd/nn/` source directory.

---

## Consequences

- Notes are durable plain Markdown files readable by any editor independent of `nn`
- Git history is a semantic audit trail of intellectual development
- The LLM can drive all operations without a TTY by using flag-only invocations
- Multiple notebooks are supported from the start via named config entries
- The `type` field at note creation enforces atomicity discipline before content is written
- Link annotations are required (schema-level enforcement), preserving the Zettelkasten's
  core value proposition: connections carry meaning, not just pointers

---

## Alternatives Considered

**Use `zk-org/zk` directly:** Feature-complete but not designed for LLM use. No `type`/`status`
fields, no annotation enforcement, interactive-first. Wrapping it would require fighting the tool.

**Folder hierarchy instead of flat directory:** Rejected. Hierarchical classification before
understanding is the failure mode the Zettelkasten method is designed to prevent.

**SQLite as primary store (no Markdown files):** Rejected for M1. Files-first means the system
survives any software discontinuity. SQLite-primary can be a later backend option.

**Timestamp ID without random suffix:** Rejected. LLM batch creation (multiple notes per
second) causes collisions with 12-digit timestamps. 14-digit + 4-digit random suffix is safe.
