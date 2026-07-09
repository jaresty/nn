---
name: nn-session-debrief
description: End-of-session review — surface what was captured, flag un-promoted drafts with new links, propose a session summary note, and update the daily page. Also supports weekly review mode (/nn-session-debrief --weekly) and on-demand daily page update (/nn-session-debrief --daily).
when_to_use: At the end of a working session to consolidate captures, assess promotion eligibility, and record what was learned. Invoke with /nn-session-debrief. For a weekly review pass (wider time window, stale draft sweep, promotion focus), invoke with /nn-session-debrief --weekly. To update only the daily page mid-session or on demand, invoke with /nn-session-debrief --daily. For a lightweight mid-session capture pass without relay/daily updates, invoke with /nn-session-debrief --partial.
---

# nn-session-debrief

End-of-session review: surface what was captured, flag un-promoted drafts with new links, and propose a session summary note.

## When to use

Invoke at the end of a working session to:
- Review what was captured
- Identify draft notes that now have enough inbound links to be promoted
- Propose a session summary note linking the session's captures

Invoke with `--daily` to update only the daily page — skips all other steps. Use mid-session or any time the user asks to update the daily page.

Invoke with `--partial` for a lightweight mid-session capture pass:
- Runs steps 1, 1b, and 2 only (review captures, surface search gaps, find promotion candidates)
- Skips step 6 (daily note / relay update) and step 7 (session summary)
- Tags each note captured or promoted in this pass with `partial-debrief` so the full end-of-session debrief can skip already-processed notes
- Use when context is filling and you want to capture findings without the full debrief overhead
- After completing steps, emit to the user: "Relay block not updated — run /nn-session-debrief to complete handoff."

**Full debrief behaviour when `--partial` runs occurred:** skip notes already tagged `partial-debrief` in steps 1 and 2; still run steps 3, 3b, 3c, 6, and 7 in full.

Invoke with `--weekly` for a broader review pass:
- Sweep drafts modified in the past 7 days
- Surface stale drafts (modified >7 days ago, still draft)
- Focus on promotion eligibility across the full recent period
- Skip session summary (weekly pass is not session-scoped)

## Workflow

> **Execution contract** — each step names its completion artifact in a `Completion:` line.
> Before starting the next step, emit a `## Checkpoint` block quoting that artifact verbatim.
> If a step is skipped, the checkpoint must read `Skipped: Step N — <reason>` (reason required;
> silent omission is a violation).
>
> Mode skip tables — each skipped step still requires a `Skipped:` checkpoint:
> - `--daily`: skips steps 1, 1b, 2, 3, 3b, 3c, 4, 5, and 7
> - `--partial`: skips steps 3, 3b, 3c, 4, 5, 6, and 7
> - `--weekly`: skips step 7
>
> Example checkpoint after step 1:
> ```
> ## Checkpoint
> Step 1 complete — reviewed: 20260707-1234, 20260707-5678
> ```
> Example skip checkpoint:
> ```
> ## Checkpoint
> Skipped: Step 3 — daily mode
> ```

### 1. Review captures

First, get today's daily note to establish the session boundary:

```
nn show daily
```

Note the `created:` timestamp from the output — this is the session boundary. Notes created at or after this timestamp are this session's captures.

List all draft notes:

```
nn list --sort created --status draft --json
```

Do not pipe this command — piping to `head` or any other command truncates the result and silently drops captures. Filter to session captures by comparing each note's `created` field against the daily note's `created:` timestamp from above. Notes created before that timestamp are pre-session drafts — skip them in step 1; they are candidates for step 2's promotion sweep.

For each session capture, run `nn show <id>` to verify the content is accurate and the title is precise.

**Completion:** emit every ID from the JSON array whose `created` value is at or after the daily note's `created:` timestamp. If no notes meet that condition, emit `"zero captures this session"` instead. (`nn show` on each ID is a required step; the completion artifact is the ID list regardless of whether every show completed.)

### 1b. Surface capture opportunities from session searches

Scan the conversation transcript for `nn list --search` calls made this session. For each:
- If the result was `[]`: the topic was searched but returned nothing — it is a capture candidate.
- If the result was non-empty but no subsequent `nn new` or `nn update` followed for those IDs: the search may have surfaced relevant notes that were read but not captured or linked.

For each zero-result search, propose a new note:

```bash
nn new --title "<topic>" --type <concept|observation|question> --content "..." --no-edit
```

For each non-empty search where relevant results were found but nothing was captured, flag the gap: "Searched for X, found Y — no capture or link recorded."

**Completion:** emit each gap flagged and each note created, or `"no zero-result searches found"` if none.

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

**Completion:** emit each promotion decision (promoted `<id>` or deferred with reason), or `"no candidates"` if none.

### 3. Capture friction points

Ask: **"What slowed you down this session? Any theories about why?"**

Also scan the conversation transcript for **repeated similar edits** — making the same kind of change in multiple places is a strong signal that the concern is not centralized and a refactor may be warranted. Flag these as friction candidates even if the user did not name them explicitly.

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

**Completion:** emit each friction note ID created, or `"no friction identified"` with a reason.

### 3b. Flag retrieved notes as potentially stale

Scan the conversation transcript for `nn show <id>` calls made this session (excluding virtual notes). For each real note that was shown, run:

```bash
nn show <id>
```

Check the `modified:` timestamp. If the note was last modified more than 14 days ago and content from the session contradicts or extends what the note says, flag it:

```
Stale candidate: <id> — <title> (modified: <date>)
Reason: session discovered [X] which is not reflected in the note body.
```

Propose an update for each flagged note. If the note content is still accurate, note "content verified — no update needed."

**Completion:** emit `"Stale candidate: <id>"` for each flagged note, or `"content verified — no update needed"` for each checked note that was accurate.

### 3c. Capture knowledge transmission opportunities

Scan the conversation transcript for moments where the user learned something new, asked for an explanation, or worked through unfamiliar territory. These are candidates for quiz notes — active recall reinforces durable knowledge.

Look for:
- Questions the user asked that Claude answered at length
- Concepts the user encountered for the first time (signaled by phrases like "I didn't know", "interesting", "makes sense now", "ah")
- Decisions where the reasoning was non-obvious and worth internalizing
- Errors or misunderstandings that were corrected — the delta is often the most learnable unit

Before proposing, generalize each candidate: strip the session-specific context and extract the transferable principle. The question should test understanding of a concept or invariant, not recall of what happened in this session. Ask: "would this question make sense to someone who didn't see this session?" If no, abstract up one level.

For each candidate, propose a `question` note tagged `quiz`:

```bash
nn new \
  --title "Quiz: <concise question>" \
  --type question \
  --tags "quiz" \
  --content "## Question\n<the question — generalized, no session-specific details>\n\n## Answer\n<the answer as a transferable principle>\n\n## Why it matters\n<the consequence of not knowing this>" \
  --no-edit
```

Present candidates to the user before creating — quiz notes are only worth capturing if the user confirms they want active recall on the topic. Ask: **"These moments looked like new knowledge — worth a quiz note?"**

**Completion:** emit each quiz note ID created, or `"no candidates"` if none were identified.

### 4. Run nn-refine on key captures

For each significant capture from this session, invoke the `nn-refine` workflow to check atomicity, links, and title quality.

**Completion:** emit each note ID passed to nn-refine, or `"no significant captures"` if none.

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

### 6. Update daily page

Maintain a running daily log note for today. This step runs in all modes including `--daily` (where it is the only step).

**Show (or create) today's daily note:**

```bash
nn show daily
```

This creates today's `Daily: YYYY-MM-DD` note if it does not yet exist (tagged `daily`, expires in 7 days). Read the output to get the current note ID and body before updating.

**Update sections — `nn update daily` bypasses `--since` and upserts automatically:**

```bash
# Replace Done with today's completed work only (do NOT include prior entries — they accumulate across days otherwise)
nn update daily --replace-section "Done" \
  --content "- <new entry 1>\n- <new entry 2>" \
  --no-edit

# Replace Open with current unresolved items
nn update daily --replace-section "Open" \
  --content "<current open items>" \
  --no-edit

# Replace Relay with current handoff state
nn update daily --replace-section "Relay" \
  --content "In progress: <what is in flight>\nNext: <immediate next action>\nBlocked: <blockers or —>" \
  --no-edit
```

If the note body has no existing sections yet, use `--content` with the full structured body instead of `--replace-section`:

```bash
nn update daily --content "## Done
- <entry>

## Decided
- <decision>

## Open
<open items>

## Relay
In progress: <what is in flight>
Next: <immediate next action>
Blocked: <blockers or —>" --no-edit
```

**Section semantics — ledger discipline:**
Each section is a categorized ledger entry. Write only content that belongs to the category — no narrative, no filler, no reasoning traces. Every entry must be self-contained: a reader with no prior context must be able to understand it without reading the conversation that produced it.

- **Done** — today's completed work only. Replace the section each session with this session's entries — do not carry forward entries from prior sessions or prior days. Each entry names what changed and where (e.g. "Added `--since` flag to `nn update` — `cmd/nn/cmd/update.go`"). Prior sessions' Done entries belong in session summary notes.
- **Decided** — decisions with enough context to reconstruct why. Name the decision, the alternative considered, and the reason chosen (e.g. "`--since` hard-required, not optional — all callers must prove they read the note before writing"). Append new decisions.
- **Open** — replace each session. Forward-looking: unresolved questions, next actions, things that need follow-up. Must be specific enough to act on without re-reading the session.
- **Relay** — replace each session. Explicit handoff state written so the next session can resume without re-reading anything: what is in flight (named artifact or task), what the immediate next action is, and what is blocked. This is relay-topology writing: schemas, invariants, and dependency relationships must be named explicitly — "the thing we discussed" is not relay-compliant.

**Link the daily page to notes created this session:**

```bash
nn bulk-link <daily-id> \
  --to <id1> --annotation "worked on this today" --type source-of \
  --to <id2> --annotation "worked on this today" --type source-of
```

**Completion:** emit the daily note ID and the sections updated (Done / Open / Relay).

### 7. Propose session summary note

Create a session summary note linking the key captures:

In weekly mode, skip this step.

```
nn new --title "Session: <date> — <topic>" --type observation --content "## What was captured\n\n..." --no-edit
nn bulk-link <summary-id> \
  --to <id1> --annotation "captured this session" --type source-of \
  --to <id2> --annotation "captured this session" --type source-of
```

**Completion:** emit the session summary note ID, or `"skipped — weekly mode"` if in `--weekly`.

## nn commands used

```
nn list --sort created --status draft --json  # no pipe — piping truncates
nn show <id>
nn backlinks <id> --json
nn promote <id> --to reviewed|permanent
nn new --title "..." --type observation --content "..." --no-edit
nn bulk-link <from> --to <id> --annotation "..." --type TYPE [--to <id> ...]
nn review                                    # weekly mode: full health report
nn review --format json
nn list --search "friction" --status draft --json   # weekly: sweep open friction notes
nn list --search "Daily: <YYYY-MM-DD>" --tag daily --json   # daily page lookup
nn new --title "Daily: <YYYY-MM-DD>" --type observation --tags "daily" --content "..." --no-edit
nn update <daily-id> --replace-section "Done" --content "..." --since <modified> --no-edit
nn update <daily-id> --replace-section "Open" --content "..." --since <modified> --no-edit
nn update <daily-id> --replace-section "Relay" --content "..." --since <modified> --no-edit
```

## Success criteria

- All captures from the session have been reviewed for accuracy
- Promotion-eligible drafts have been promoted or a reason given for deferral
- Friction points from the session have been captured as `hypothesis` or `observation` notes tagged `friction`
- Knowledge transmission opportunities have been proposed to the user; accepted ones captured as `question` notes tagged `quiz`
- Today's daily page exists and has been updated with Done/Open/Relay reflecting the current session
- A session summary note exists linking the key captures, or a reason given for skipping it
