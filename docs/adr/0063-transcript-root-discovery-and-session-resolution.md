# ADR-0063: Transcript root discovery and session resolution

Status: Accepted

`nn transcript` discovers sessions from a bounded registry of supported agent
roots when `ls`, `scan`, or `search` receives no explicit path. The initial
registry supports Claude (`~/.claude/projects`), Codex
(`${CODEX_HOME:-~/.codex}/sessions` and `archived_sessions`), and Pi
(`~/.pi/agent/sessions`). Provider-specific environment overrides are honored
only where verified from the provider's own documentation or source.

Explicit paths remain the highest-priority, byte-preserved, backward-compatible
operand so retained snapshot identity does not change. Other single-session
operands resolve against the discovered inventory: a unique session ID selects
the exact canonical path retained by that inventory;
zero matches fail as not found and multiple matches fail as ambiguous with
stable candidate diagnostics. Resolution never reconstructs a path from a
project name, sanitized working directory, or session naming convention.

Keep three responsibilities separate: the provider registry supplies bounded
candidate roots, inventory discovery retains canonical existing paths and
provenance, and schema adapters parse selected transcripts. Commands resolve
at their CLI boundary and continue passing paths to existing transcript logic.
Search remains corpus-oriented: explicit inputs define its corpus, while no
inputs select the available registered roots.

Discovery must disclose unavailable roots without preventing available
providers from working. Cursor, snapshot, replay, pagination, and retained-path
checks bind canonical paths and inventory metadata rather than shorthand input.
An inventory change or ambiguous identifier therefore cannot silently select
different evidence.

Primary location evidence:

- Claude Agent SDK stores transcripts under `~/.claude/projects/` by default
  (Claude Agent SDK session-storage documentation).
- OpenAI Codex source `codex-rs/utils/home-dir/src/lib.rs` honors `CODEX_HOME`
  and otherwise uses `~/.codex`; `codex-rs/rollout/src/lib.rs` defines
  `sessions` and `archived_sessions` rollout subdirectories.
- Pi session-format documentation stores sessions beneath
  `~/.pi/agent/sessions/`, grouped by working directory.

Rejected alternatives are recursively scanning the home directory, guessing
paths from identifiers, selecting the first ambiguous match, and duplicating
root or resolution policy independently in each command.
