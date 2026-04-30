# ADR 0013: Tag Intelligence

## Status

Accepted

## Context

Tags are currently applied manually by the author (human or LLM) at note creation or update time. There is no way to enumerate the existing tag vocabulary, and no mechanism to suggest tags based on similarity to existing notes. This leads to tag drift — new notes use idiosyncratic tags that don't match the established vocabulary, weakening cluster coherence over time.

The primary user is an LLM agent. The design must minimize tool calls while surfacing actionable signals.

## Decision

### 1. `nn tags` command

A new top-level command that returns all tags in the notebook with note counts.

```
nn tags [--json]
```

Default output (plain):
```
hooks          12
architecture    8
protocol        5
```

JSON output:
```json
[
  {"tag": "hooks", "count": 12, "notes": ["id1", "id2", ...]},
  ...
]
```

Use case: an LLM agent runs `nn tags` before tagging a new note to orient itself against the existing vocabulary.

### 2. `nn suggest-tags <id>`

A new command that returns tag suggestions for a given note, derived from BM25-similar notes that share tags the target note lacks.

```
nn suggest-tags <id> [--json]
```

Plain output:
```
hooks          (from 3 similar notes: id1, id2, id3)
architecture   (from 2 similar notes: id1, id4)
```

JSON output:
```json
[
  {"tag": "hooks", "from_notes": ["id1", "id2", "id3"]},
  {"tag": "architecture", "from_notes": ["id1", "id4"]}
]
```

Only tags that appear in ≥2 similar notes are surfaced. The LLM decides whether to apply them via `nn update <id> --tags`.

### 3. Post-write suggestions in `nn new` and `nn update`

After writing and committing a note, `nn new` and `nn update` print advisory suggestions to stdout. These are not part of the commit and do not affect the note state.

```
Created: 20260430-1234-my-note.md

Suggestions:
  links: 20260417-5678 "Hook latency tradeoffs" (0.82), 20260419-9012 "Stop hook design" (0.71)
  tags:  hooks, architecture (from 3 similar notes)
```

A `--no-suggest` flag suppresses this output for scripts and pipelines that don't want it.

Suggestions are printed to stdout only when stdout is a TTY or `--json` is not active. When `--json` is active, suggestions are included in the JSON response under a `suggestions` key alongside the created/updated note metadata.

### 4. Stop-hook agent update

`nn-stop-agent.md` gains a tag-suggestion step in Phase 2: after checking links via `nn suggest-links`, run `nn suggest-tags <id>` and apply tags that appear in ≥2 similar notes without asking — tags are low-risk and easily removed.

## Consequences

- LLM agents can orient against the existing tag vocabulary before writing (`nn tags`)
- Tag drift is reduced: post-write suggestions surface vocabulary mismatches immediately
- No extra tool calls required to see suggestions — they arrive with the write response
- `--no-suggest` preserves scripting ergonomics
- `nn suggest-tags` is composable: usable standalone or as part of the stop-hook debrief
- Tag application in the stop-hook agent is autonomous (no confirmation) — acceptable because tags are non-destructive and reversible
