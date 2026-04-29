# ADR 0012: Note Access Telemetry

## Status

Accepted

## Context

The nn promotion workflow requires human or agent judgment to identify which draft notes are ready to promote. The current criteria (atomic body, inbound link, durable claim) are structural — they describe what a promotable note looks like, but don't surface notes that are *worth* promoting based on observed use.

A note retrieved via `nn show` but never followed up on (no link, no edit, no promotion) is a signal: the note was relevant enough to retrieve but not acted on. Over multiple sessions this pattern identifies notes that are either under-linked, under-refined, or waiting for a connection that hasn't been made yet.

## Decision

Add an append-only access log at `~/.config/nn/access.log`. It records deliberate note retrievals and is advisory only — it never affects correctness of any core operation.

### Log format

One line per event:

```
<RFC3339 timestamp> show <note-id>
```

Example:

```
2026-04-29T13:05:00Z show 20260401-1234
2026-04-29T13:07:00Z show 20260401-5678
```

### What writes to the log

- `nn show <id>` — deliberate retrieval of a specific note

`nn list --search` is explicitly excluded. Search results are scanned, not retrieved — too noisy to be a useful signal.

### What reads the log

- `nn list --stale` — returns notes accessed in the log within the last N days (default 7) with no git commit touching that note's file since the last access timestamp. These are candidates for follow-up.

The stop-hook agent (`nn-stop-agent.md`) gains a new step: run `nn list --stale --json` and for each result, propose `nn link` or `nn update` as appropriate.

### What the log is NOT

- Not ground truth — a note absent from the log is not less valuable
- Not a promotion gate — access count does not factor into `nn promote` eligibility
- Not synced or backed up — loss of the log is acceptable; it is rebuildable from git log timestamps as an approximation
- Not read by any command that affects note state

## Consequences

- `nn show` gains a side effect (log write). It is fire-and-forget: if the write fails, `nn show` still succeeds.
- A new `--stale` flag on `nn list` enables access-informed surfacing without changing the core data model.
- The stop-hook agent becomes more targeted: instead of scanning all recent drafts, it can focus on notes the user already retrieved but didn't act on.
- Access log grows unboundedly. A future `nn log prune` command can truncate entries older than N days, but is not required for correctness.
