---
name: nn-session-debrief
description: End-of-session review — surface what was captured, flag un-promoted drafts with new links, and propose a session summary note. Also supports weekly review mode (/nn-session-debrief --weekly).
when_to_use: At the end of a working session to consolidate captures, assess promotion eligibility, and record what was learned. Invoke with /nn-session-debrief. For a weekly review pass (wider time window, stale draft sweep, promotion focus), invoke with /nn-session-debrief --weekly.
---

# nn-session-debrief

End-of-session review: surface what was captured, flag un-promoted drafts with new links, and propose a session summary note.

## When to use

Invoke at the end of a working session to:
- Review what was captured
- Identify draft notes that now have enough inbound links to be promoted
- Propose a session summary note linking the session's captures

Invoke with `--weekly` for a broader review pass:
- Sweep drafts modified in the past 7 days
- Surface stale drafts (modified >7 days ago, still draft)
- Focus on promotion eligibility across the full recent period
- Skip session summary (weekly pass is not session-scoped)

## Workflow

### 1. Review captures

List recently created notes to see what was captured this session:

```
nn list --sort created --status draft --json | head -20
```

For each captured note, run `nn show <id>` to verify the content is accurate and the title is precise.

### 2. Find promotion candidates

Find draft notes that may be ready for promotion:

```
nn list --type observation --status draft --json
nn list --type concept --status draft --json
```

**Promotion threshold** — promote a draft if it meets any of:
- Has reviewed inbound links and a focused body
- Has 2+ outbound links and a focused body (well-connected even if not yet referenced)

For each candidate, check link counts:

```
nn backlinks <id> --json
```

```
nn promote <id> --to reviewed
```

### 3. Capture friction points

Ask: **"What slowed you down this session? Any theories about why?"**

For each friction point identified, capture it as a note:

```bash
nn new --title "Friction: <short description>" \
  --type hypothesis \
  --tags "friction" \
  --expires-when "resolved or disproved" \
  --content "## Observed\n<what happened>\n\n## Theory\n<why this might be happening>" \
  --no-edit
```

Use `hypothesis` type when there is a theory to test; use `observation` when the friction is documented but the cause is unclear. Always tag `friction` so weekly review can surface the full backlog with `nn list --search friction`.

Link friction notes to relevant protocol or concept notes they implicate:

```bash
nn link <friction-id> <related-id> --type questions --annotation "this friction challenges whether the approach works"
```

In weekly mode, sweep existing friction notes for resolution:

```bash
nn list --search "friction" --status draft --json
```

For each: is it resolved? Promote to `reviewed` with a note on what changed, or update the body with new evidence.

### 4. Run nn-refine on key captures

For each significant capture from this session, invoke the `nn-refine` workflow to check atomicity, links, and title quality.

### 5. (Weekly mode only) Notebook health review

When invoked with `--weekly`, run the full health report first:

```
nn review
```

Use its output to drive the rest of the pass:
- **Orphans section** → surface each orphaned draft for linking or deletion
- **Long notes section** → flag notes exceeding the atomicity threshold for splitting
- **Aging notes section** → two buckets using the same thresholds as `nn show` freshness:
  - `aging (3–14 days)`: surface for recheck — propose `nn show <id>` and update if content is stale
  - `stale (>14 days)`: content may be outdated — verify before relying on it; propose update or deletion if superseded
- **Structural gaps** → act on any other pattern `nn review` identifies

Then sweep unactioned drafts:

```
nn list --older-than 14 --status draft --rich --json
```

`--rich` includes `created` and `modified` timestamps in JSON output. `--older-than 14` filters to notes not modified in 14+ days (same threshold as the stale tier). `--sort modified` sorts by last-modified date. These flags compose freely.

For each draft not touched in 14+ days: check outbound link count first.
- **Has outbound links + coherent body** → promotion candidate, not a deletion candidate; propose `nn promote`
- **Zero links in either direction** → true orphan, candidate for deletion; surface for user to decide
- **No inbound, but has outbound** → dead-end note that hasn't been linked to yet; link it rather than delete it

**Available flags on `nn list` — use these before reaching for workarounds:**
- `--sort modified` — sort by last-modified date (most-recent first)
- `--older-than N` — notes not modified in the last N days
- `--rich --json` — includes `created` and `modified` timestamps in JSON output
- `--no-inbound` — notes with no backlinks (deletion candidates; stricter than `--orphan`)
- `--orphan` — notes with zero links in either direction

### 6. Propose session summary note

Create a session summary note linking the key captures:

In weekly mode, skip this step.

```
nn new --title "Session: <date> — <topic>" --type observation --content "## What was captured\n\n..." --no-edit
nn bulk-link <summary-id> \
  --to <id1> --annotation "captured this session" --type source-of \
  --to <id2> --annotation "captured this session" --type source-of
```

## nn commands used

```
nn list --sort created --status draft --json
nn show <id>
nn backlinks <id> --json
nn promote <id> --to reviewed|permanent
nn new --title "..." --type observation --content "..." --no-edit
nn bulk-link <from> --to <id> --annotation "..." --type TYPE [--to <id> ...]
nn review                                    # weekly mode: full health report
nn review --format json
nn list --search "friction" --status draft --json   # weekly: sweep open friction notes
```

## Success criteria

- All captures from the session have been reviewed for accuracy
- Promotion-eligible drafts have been promoted or a reason given for deferral
- Friction points from the session have been captured as `hypothesis` or `observation` notes tagged `friction`
- A session summary note exists linking the key captures, or a reason given for skipping it
