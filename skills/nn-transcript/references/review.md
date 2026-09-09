---
name: review
applies_when: "When entering the default Awaiting return Office view, inspecting unclosed handoffs, opening the Unclosed Work Desk, finding deterministic patterns in one retained review population, or preparing course corrections."
---

# Transcript Review Queue / Unclosed Work Desk

Use the selected discovery row's exact canonical path. The CLI performs no LLM work.
Load `nn skills get nn-transcript --reference interaction` for action targeting, inspection envelopes,
retained view records, and Back. This reference owns desk population and evidence-guided Find.

```bash
nn transcript review <session> --queue awaiting-return --order observed-recent --limit 20 --json
nn transcript review <session> --queue open-handoff --order observed-recent --limit 20 --json
nn transcript review <session> --queue ambiguous-handoff --order canonical --limit 20 --json
nn transcript review <session> --queue archive --limit 20 --json
```

Initial support is Pi only; other schemas fail explicitly, never return a misleading empty queue.
`archive` means all retained non-ROOT rooms, not completed work. Queue metadata requires `--json`;
for readable evidence use `--last N --format text`.
`--limit` is 1–200, default 20. Continue with `--cursor <next_cursor>` and identical options,
including limit. Changed selection rejects the cursor. Live source appends do not: continuation uses
the retained capture without rereading live files. Refresh deliberately by omitting cursor/snapshot.
Each source is captured once at its observed byte length, excluding an unfinished final record.
Files are captured sequentially, not at a globally simultaneous instant. Captures are private local
cache artifacts (24-hour retention); missing, expired, or corrupt required cache artifacts require explicit refresh.
Bundle transport pages are retained separately and replay without reloading raw captures or recomputing
selection. Advancing a room cursor still requires its raw capture.
`capture_id` identifies the retained inputs. Canonical path and authenticated ownership remain unchanged.

## Authority and counts

- `awaiting-return`: authenticated launch count ≥1 and parent-return count 0, regardless of terminal count.
  This is the default Pi Office population, across all retained non-ROOT rooms, not only ROOT children.
  Separate **No terminal recorded** (`terminals == 0`) from **Terminal recorded; return missing**
  (`terminals > 0`), using row badges so observed-recent ordering stays intact. Do not infer either
  condition from prose or producer status. The current Pi projection counts producer terminals as
  parent returns too, so terminal-only rows may be absent; do not manufacture them. Multiple launches
  plus a return remain ambiguous, not unmatched attempts inferred by subtracting counts.
- `open-handoff`: authenticated launch count ≥1, terminal count 0, parent-return count 0.
  Authentication means the existing exact unique owner/call-ID invocation join succeeded.
- `ambiguous-handoff`: retained launches and parent returns both exist. Their occurrence relationship
  is not authenticated; never subtract counts or pair them into attempts. This queue may include
  unauthenticated launches, explicitly distinguished by `authenticated_launches`.
- `launches`, `parent_returns`, and `terminals` are retained record counts. Pi producer records serve
  as both parent returns and terminal evidence; that does not create launch-to-return pairing.
  All producer record statuses count as terminal evidence, including unfamiliar statuses.
- Launch occurrence numbering is independent of returns. `launch_occurrence` identifies the launch
  supplying the displayed description; zero means no launch-derived label. Use **handoffs** for
  bounded occurrence retrieval rather than embedding unbounded occurrence payloads in cards.
- `pairing` and `liveness_status` are `not_inferred`. Open does not mean running, waiting, or stuck.
- `population` counts all non-ROOT rooms. `eligible` counts queue-and-pattern matches.
  `eligible = offset + returned + omitted`; offset counts earlier rows, omitted counts later rows.
- `unknown` counts rooms in the entire population with unavailable work, unknown work timestamps,
  or no authenticated launch. It can overlap eligible rooms; it is not an extra population bucket.
  Zero observed occurrences does not establish source completeness.

Labels prefer latest nonempty recorded launch description, then first owned user instruction, then
`Untitled room · <short-id>`. Whitespace folding and 120-character truncation are deterministic;
`label_provenance` identifies the source (`recorded`, `opening`, `untitled`), not LLM interpretation.
`label_event_id` addresses that source; untitled uses `unavailable`. Never treat an earlier launch's
label as proof of the assignment of a later unmatched attempt.

Work recency includes owned assistant and tool-result records, excludes user and lifecycle-only
records, and uses the greatest valid absolute timestamp. Message timestamps are a fallback; numeric
message timestamps are milliseconds. Unknown times sort last; ties use canonical agent ID.
`last_observed_event_id` identifies the ledger message containing that work (including tool blocks).
Timestamp, kind, and `recency_basis: work` describe retained observation, not process activity.

## Desk actions

Default to **Inspect recent work**, **Find patterns…**, **More…**, with **Back** and
**End** visible. Promote stronger evidence-based suggestions under the core's suggested-action contract;
keep displaced controls and **Capture…** under More. Promote **Draft correction…** only for a supported
concern after adequate assignment inspection, not on an empty result. A useful supported finding may promote
**Capture this insight**, which opens a proposal rather than writing a note. Show every returned row on the current page;
never reduce the awaiting-return population to three standout cards. Display **Awaiting return · N total ·
rows X–Y**, using eligible/offset/returned, and expose Next when `next_cursor` is nonempty. State later-row
omissions explicitly; a page is not the full list. If eligible is zero, say no authenticated launches
without recorded returns were found in retained evidence—not that no workers are running. At most three
suggested actions does not limit listed rooms. Keep **More → All rooms**, Back, and End visible. Keep exact
picker labels. Position may encode observation time only when the legend says so; do not use red/green
as an implicit stuck/running signal. Hierarchy and archive remain reachable under More….

Back restores retained rows, options, snapshot, question, and inspected-room set without rereading.
Refresh reacquires evidence and preserves the chosen lens unless unsupported.

## Read bundle contents without scripts

```bash
nn transcript review <session> --queue open-handoff --limit 3 --last 5 --format text
```

Prefer this for a readable bounded scan. It includes payloads internally, consumes every retained
transport page, reconstructs segmented records, and renders room headings and source event IDs—no
Python, `jq`, or manual page loop. It does not advance the room cursor. `--max-text-chars` bounds each
event (default 1000, 1–10000); `--max-output-chars` bounds the complete display (default 24000,
2048–200000). Counts distinguish earlier/other-room omissions from events omitted by the display cap.
Truncation markers disclose shortened text; hidden thinking/signatures are not rendered. Failure
fields remain visible. This is lossy deterministic rendering, not semantic summarization.

Supply the text output's `--snapshot` with identical selection options to replay or adjust display
limits over the same capture. Text rejects explicit `--json`, `--payload`, and `--page`; `--last` is
required. Use JSON for exact evidence when a conclusion depends on clipped text. Failed later transport
pages produce no partial text. Report display truncation as an inspection limit, not complete semantic
coverage merely because transport is complete.

## Assignment-aware initial inspection

When the question concerns work relative to assignments across several rooms, retrieve both initially:

```bash
nn transcript review <session> --queue open-handoff --limit 3 --last 5 --include-assignment --format text
```

`--include-assignment` requires `--last`, implies payloads (rejects `--payload=false`), and works with
JSON or text. It reuses context's handoff join on the same captured sources for every selected room:
all independent launch occurrences, not a chosen governing attempt. Steering remains unavailable;
missing/ambiguous joins are explicit. Text labels launch records and their qualification. JSON adds
`include_assignment` and `selected_assignments` to the receipt, and `launches`, `assignment_events`,
`steering_status`, and `governing_attempt` to each room. Launch events retain IDs and `section: launch`.
`selected_events` continues to count recent events only; assignments are counted separately.

The option participates in snapshot binding. Repeat it for replay/pages; default activity-only bundles
and their cache bindings remain unchanged. Assignment payloads can be large or unavailable: preserve
truncation/coverage and budget them initially instead of consuming one unplanned follow-up per room.
For one selected room, prefer **context** directly. Do not include assignments in ordinary activity-only
scans merely because the command supports them. **interaction** owns question-based authorization.

## Find: evidence-guided discovery

Find is an intent, not a compulsory filter chooser. Resolve the current room/desk and inspection
envelope through **interaction**. If the action already names approved room, sample, and follow-up
bounds, selecting it executes without another confirmation. Otherwise offer one concrete bounded
inspection action, not a list of technical pattern categories. Preserve active filters; an empty
filter never silently falls back to the underlying desk.

Use `review --last N --format text` for activity-only inspection; add `--include-assignment` when
alignment is the question and assignments are planned into the initial budget. Explain supported observations,
separate candidate patterns from conclusions, and nominate one useful exact event/assignment when
needed. Within the envelope, retrieve complete exact-event evidence with `events --event ID --payload
--json`, following transport rules. Stop at sufficient evidence, exhaustion, or unavailability. Do not
infer a blocker from a failed command before inspecting its cause/recovery, or poor productivity from
read-heavy work. Repeated message/result representations are not repeated attempts. Say 'nothing
compelling in this sample' when appropriate; capture useful discoveries as approval-only proposals.

The following deterministic quick patterns are optional native filters over the chosen queue, not
semantic diagnoses or a mandatory menu:

```bash
nn transcript review <session> --queue open-handoff --pattern repeated-tools --limit 20 --json
```

Choices: `errors`, `interruptions`, `repeated-tools`, `repeated-commands`, `timing-gaps`,
`missing-evidence`. Every result discloses `algorithm`. Commands match exact argument strings without
shell normalization. Timing gaps use a disclosed 300-second threshold and imply no cause.
A new pattern selection acquires a new snapshot; disclose this rather than claiming the old desk
snapshot was reused. Never silently widen the queue or selected session.

Semantic questions (instruction drift, shared blockers, duplicated investigation) are LLM-owned.
Use the approved inspection envelope, distinguishing the initial sample from bounded follow-up.
If only a last-5-event window was approved, do not invent additional authorization. Selecting a concrete
bounded proposal is its approval, not the start of another confirmation loop. Prefer the readable native
bundle above; lossless JSON remains available when the evidence requires it:

```bash
nn transcript review <session> --queue open-handoff --limit 3 --last 5 --payload --json
# Same options plus --page <next_page> --snapshot <snapshot> for EVERY remaining transport page.
```

The first record (`kind: review_receipt`) carries the review snapshot, population/eligible counts,
room offset, retrieved/omitted/unavailable room counts, selected/omitted-earlier event counts, and
`next_room_cursor`. Each `review_room` record carries its exact row, detail availability, query receipt,
and room snapshot; source events have `section: recent` and retain original event IDs/ordinals.
Metadata records can also fragment: reconstruct every ordered segment under **events** transport rules.
Preserve stream order; original ordinals are room-local, not a cross-room sort key.

`--limit` bounds the room selection; `--last` bounds events per room (1–200). `--page` advances transport,
not rooms. Only after all transport pages are complete may `next_room_cursor` advance room selection.
Keep all options identical, including payload, last, limit, queue, order, and pattern. Continuation reads
the retained capture, even after live source deletion. Receipts disclose `capture_id`,
`capture_boundary: sequential_complete_record_prefixes`, and `capture_sources` (source count).
Detailed source prefix bytes/digests reside in the private capture manifest under the OS user-cache
`nn/transcript-captures-v1/<capture_id>.json`; they are not repeated in the default receipt.
A fresh bundle reacquires the selected queue: compare its IDs with the approved
room set before interpreting; disclose changes and reconfirm rather than silently extending scope.

Report eligible, inspected, uninspected, omitted, and unknown coverage. CLI counts describe retrieval,
not LLM inspection (`inspection_status: not_inferred`). Cite event IDs and mark findings interpreted.
Do not claim globally atomic original-source completeness or scan rooms outside the approved population.

## Correction drafts

Load **context** and retrieve `nn transcript context <session> <agent-id> --last 5 --json` to obtain
recorded launch assignments and bounded recent evidence in one paginated bundle. **handoffs** remains
the owner for detailed occurrence semantics; context does not select a governing launch or infer steering.
Disclose omitted earlier history and cite event IDs. Expand only when needed and with disclosed scope.
Draft proposals only; do not send, steer, or stop without separately authenticated runtime authority.
Shared corrections must name each applicable room and preserve uncertain matches/counterexamples.
