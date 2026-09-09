---
name: review
applies_when: "When inspecting unclosed handoffs, opening the Unclosed Work Desk, finding deterministic patterns in one retained review population, or preparing course corrections."
---

# Transcript Review Queue / Unclosed Work Desk

Use the selected discovery row's exact canonical path. The CLI performs no LLM work.

```bash
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

Expose **Inspect recent work**, **Draft correction…**, **Find patterns…**, **More…**, with **Back** and
**End** visible. Show at most three standout cards and explicit displayed/omitted counts. Keep exact
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

## Find patterns…

Deterministic quick patterns are native filters over the chosen queue:

```bash
nn transcript review <session> --queue open-handoff --pattern repeated-tools --limit 20 --json
```

Choices: `errors`, `interruptions`, `repeated-tools`, `repeated-commands`, `timing-gaps`,
`missing-evidence`. Every result discloses `algorithm`. Commands match exact argument strings without
shell normalization. Timing gaps use a disclosed 300-second threshold and imply no cause.
A new pattern selection acquires a new snapshot; disclose this rather than claiming the old desk
snapshot was reused. Never silently widen the queue or selected session.

Semantic questions (instruction drift, shared blockers, duplicated investigation) are LLM-owned.
Confirm a bounded plan naming the eligible IDs, number of rooms, and last-5-event window. Use the native
bundle instead of an agent loop or a Python collector:

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
