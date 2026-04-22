# ADR-0011: LLM-Ergonomic Update Primitives

**Status:** Accepted — pending implementation
**Date:** 2026-04-21
**Authors:** jaresty

---

## Context

`nn` is designed for programmatic use by an LLM as its primary user. Despite this, LLMs
consistently fall back to editing note files directly (via Read/Edit/Write tools) rather than
using `nn update` and related commands. A structural analysis identified five root causes:

1. **ID lookup friction** — `nn update` requires a note ID; obtaining it requires a prior
   `nn list` or `nn search` call. `nn show` already accepts title substrings; mutating
   commands do not. Every update via nn is at minimum two commands; file editing is one.

2. **Full-body reconstruction cost** — `nn update --content` replaces the entire body. An
   LLM making a targeted change must read the full body, reconstruct it with the edit applied,
   and pass the result as a flag value. The Edit tool does surgical in-place replacement with
   no reconstruction burden.

3. **Shell escaping cost** — Passing multiline markdown with backticks, quotes, and special
   characters through `--content` is error-prone. LLMs have learned that shell argument
   escaping fails unexpectedly. File editing via Write/Edit sidesteps this entirely.

4. **Missing frontmatter primitives** — `--status` is not writable via `nn update` (only
   `nn promote`, which is one-directional). `--tags` replaces all tags, requiring a
   read-before-write round trip for additive changes. Fields without flags are unreachable
   through nn.

5. **No structural cost to bypass** — Nothing fails or warns when an LLM edits files
   directly. The shortcut costs nothing at the moment of choice.

The result: LLMs use nn for creation (`nn new`) and reads (`nn show`) — where there is no
file-editing equivalent — but bypass it for updates, where the file path is lower friction.

---

## Decisions

### 1. Title-as-ID in all mutating commands

`nn update`, `nn link`, `nn unlink`, `nn promote`, and `nn delete` accept `<id-or-title>`
the same way `nn show` already does. If a title substring matches exactly one note, it is
used. If it matches multiple notes, the command lists candidates and exits with an error.

This is the single highest-leverage change — it eliminates the mandatory
`nn list → copy ID → run command` two-step for all update workflows.

### 2. `nn update --stdin`

When `--stdin` is passed, `nn update` reads the note body from stdin instead of `--content`.
This allows heredoc and pipe patterns that avoid shell escaping entirely:

```bash
nn update <id-or-title> --stdin --no-edit <<'EOF'
Full replacement body with backticks, quotes, and special chars — no escaping needed.
EOF
```

`--stdin` and `--content` are mutually exclusive. `--from-stdin` already exists on `nn new`;
this mirrors that convention.

### 3. `nn update --replace-section <heading>`

Replaces only the named markdown section in the note body. The heading is matched
case-insensitively against level-2 headings (`## Heading`). The section content is replaced
with the value of `--content` or `--stdin`. The rest of the body is preserved.

```bash
nn update <id-or-title> --replace-section "Why" --content "New explanation." --no-edit
```

If the heading is not found, the command returns an error rather than silently appending.
This directly competes with the Edit tool's surgical replacement — same operation, but routed
through nn so the change is git-committed under nn's commit convention.

### 4. `nn update --status <status>`

Makes note status directly writable via `nn update`, accepting `draft`, `reviewed`, or
`permanent`. `nn promote` remains available for the forward-only promotion workflow but is
no longer the only path to status changes.

```bash
nn update <id-or-title> --status draft --no-edit   # demote or reset
nn update <id-or-title> --status permanent --no-edit
```

### 5. `nn update --tags-add <tag>` / `--tags-remove <tag>`

Additive and subtractive tag operations that do not require knowing the current tag list.
`--tags` (full replacement) remains available. `--tags-add` and `--tags-remove` compose:

```bash
nn update <id-or-title> --tags-add "zettelkasten" --tags-remove "inbox" --no-edit
```

Both flags are repeatable for multiple tags in a single command.

---

## Out of scope

- **Arbitrary frontmatter writes** (`--field <key> <value>`) — too open-ended, breaks schema
  guarantees enforced by the note struct.
- **Pre-commit hooks blocking direct file edits** — better to reduce friction on the nn path
  than add friction to the bypass path. Reconsidered if these changes prove insufficient.
- **`nn update --append-line` / `--insert`** — line-level surgery is handled by
  `--replace-section` at a more meaningful granularity.

---

## Implementation Sequence

1. **Phase 1**: Title-as-ID in mutating commands — unblocks all other improvements, zero
   schema change, reuses existing `resolveNote` helper from `nn show`.
2. **Phase 2**: `nn update --stdin` — mirrors `nn new --from-stdin`, low implementation cost.
3. **Phase 3**: `nn update --replace-section` — requires section parsing logic.
4. **Phase 4**: `nn update --status` — trivial flag addition, replaces `nn promote` as
   primary status-setting path for LLM workflows.
5. **Phase 5**: `nn update --tags-add` / `--tags-remove` — requires tag set merge logic.

Each phase is independently shippable.

---

## Consequences

- LLM update workflows collapse from two commands to one for the common case.
- Multiline content can be passed without shell escaping via `--stdin`.
- Targeted section replacement is possible without full-body reconstruction.
- The nn path becomes strictly lower friction than direct file editing for all common
  LLM editing operations.
- `nn show` and `nn update` share the same ID resolution behaviour — consistent mental model.
- `nn promote` is not removed; it remains the human-facing forward-promotion command.
