---
name: events
applies_when: "Before targeted tree projection, readable/JSON show, event facets or payloads, exact-event retrieval, exports, failure filters, or timestamp windows."
---

# nn-transcript / events

## Targeted metadata and complete exports

For an already selected agent, use `nn transcript tree <session> --agent <id> --json`, optionally
`--fields id,type,status,cost,subtree_cost,evidence_scope`. Selectors require JSON; unknown agents,
unknown fields and empty selectors reject. Selection happens after whole-tree validation/rollup,
so subtree totals do not shrink to the selected row. Explicitly requested absent optional fields
are null. Without selectors the existing output is unchanged.

Plain `show` already returns complete text. `nn transcript show <session> <id> --all --json`
returns `{all:true,snapshot,mode,text}`. `nn transcript events <session> <id> --all` returns complete
unfragmented event objects with `all:true`, page/pages=1 and next_page=0. These are **UNBOUNDED**
exports: pipe to a JSON consumer or redirect to a file when large; do not assume they fit an agent
response. Both reject explicit --page/--snapshot flags, and show --all requires --json. The snapshot
is the same as the equivalent bounded projection. --all changes transport, not source completeness.
Default bounded modes remain unchanged and omit the all field.

## Agent selection and assignment-inclusive inspection

Use `events <session> --agent <id>` or the compatible positional `events <session> <id>`;
never both. Missing or empty agent selectors fail. Both forms bind the same identity and snapshot.
For ordinary worker inspection, include its recorded assignment in the initial acquisition:

```bash
nn transcript events <session> --agent <id> --last 10 --include-assignment --format text
nn transcript events <session> --agent <id> --last 10 --include-errors 10 --include-assignment --format text
nn transcript events <session> --agent <id> --search 'preparation' -C 3 --include-assignment --format text
```

`--include-assignment` leaves event filters/limits unchanged. Launch occurrences are a separate
section with authenticated joins, never an inferred governing task. Pi uses one shared sequential
source-prefix capture. Unsupported adapters disclose assignment support as unavailable without
suppressing event evidence. Assignment text has an independent --max-assignment-chars budget
(default 8000); clipping/omitted records are explicit. Combined text stays within 200,000 bytes.
The opt-in JSON bundle starts with an `nn.transcript.assignment-events/v1` receipt and section-tagged
events; normal 48KB pages and ordered segmentation remain lossless. Retrieve every required page.
--snapshot replays retained pages without reopening sources; preserve selection/payload options.
Omit snapshot for Refresh. Assignment inclusion rejects summary, handoff --at and --all; existing
include-errors/search incompatibilities remain. Bare events and standalone context remain supported.
Use bare events for deliberately event-only inspection, not as a prerequisite to getting assignments.

## Compact readable tails

```bash
nn transcript events <session> <agent-id> --last 8 --format text --max-text-chars 1000
```

Use this instead of `--all | jq` to obtain a readable recent tail. It selects the last N ledger events
before rendering in canonical order; N is 1–200 (required for this tail form). Includes message, tools, and lifecycle
facets automatically. Message records and their tool-block events remain separate ledger events,
not deduplicated turns. Assistant text, tool calls, results, and lifecycle status carry exact event IDs.
The header carries the snapshot, returned count, omitted event count, and detail availability.
Whitespace/control characters are flattened; each event is limited to 1–10000 readable characters
(default 1000), with explicit `[truncated N chars]` markers. This is a lossy display, not an LLM summary.
Use exact-event JSON with payload for complete evidence. Text mode rejects explicit JSON, select,
payload, all, page, snapshot, at, and summary flags. Exact `--event` text retrieval is also supported. Time and error filters remain available and
apply before last-N selection. Default JSON is unchanged. Text snapshots are evidence identifiers,
not a paging interface; refresh reruns the command.

## Recent events and failures in one acquisition

```bash
nn transcript events <session> --agent <agent-id> --last 10 --include-errors 10 --include-assignment --format text --max-text-chars 1400
```

When Refresh needs both recent activity and explicit failures, use this combined command instead
of separate ordinary and `--errors-only` reads. Each tail has its own 1–200 event limit; the failure
tail searches the whole selected ledger, not just the recent tail. Time bounds apply to both.
Both sections share one composite snapshot and identify overlapping event IDs. IDs and canonical
order remain unchanged; empty failure tails are explicit. Older failures do not establish unresolved
problems, and no failures is not a health verdict. These are events, not deduplicated work messages.

The shortcut is text-only and rejects errors-only, search/context, exact-event, summary, handoff,
and JSON/pagination options. Existing per-event clipping applies, with an atomic 200,000-byte
combined output limit. One ledger acquisition is not an atomic multi-source capture. The snapshot
identifies evidence, not retained replay; Back uses retained output and Refresh acquires anew.

## Search and grep-style context

```bash
nn transcript events <session> <agent-id> --kind tool_call --search 'bar build' -B 2 -A 8 --format text
nn transcript events <session> <agent-id> --event <event-id> -C 5 --format text
nn transcript events <session> <agent-id> --role assistant --search 'decision' --last 5 --format text
```

`--kind` selects message/tool_call/tool_result/lifecycle. `--role` matches the native role of
**message events only**. `--search` matches case-insensitive literal readable text, tool names and
decoded arguments/results, or lifecycle status—not thinking, signatures or unrelated metadata.
`--regex` explicitly enables Go regex (case-sensitive unless `(?i)` is supplied). Filters combine
with AND, including time/error filters. Exact `--event` cannot combine with anchor filters.

`-B/--before-context N`, `-A/--after-context N`, and `-C/--context N` count ledger **events**, not
messages/turns. Bounds are 0–200; -C rejects combination with -A/-B. Context ignores anchor filters
(including time/error) and stays in the selected agent's canonical ledger. For “after completion,”
use the uniquely matched result's ID, not the call ID; absent/ambiguous tool joins remain explicit.

The first 20 matching anchors are selected by default; `--max-matches 1..200` changes this cap.
`--last N` selects the most recent matching anchors, still capped by --max-matches. No anchor
filters means all events are eligible. Overlapping/adjacent windows merge. `window_match` marks
selected anchors (other events, even those satisfying the predicate beyond the cap, are context);
`window_start` marks each merged window. Text prints MATCH/CONTEXT and inter-window omission counts.
`query.window` reports matched/selected/omitted anchors, context count and number of windows.
Time/error exclusion counters describe rejected **anchors**, not a partition of returned context.

At most 2,000 expanded events and 200,000 rendered text bytes; larger requests fail with reduction
guidance. Per-event text clipping remains explicit. JSON retains normal 48KB pagination and
segmentation; retrieve all relevant pages/segments. The full matching input, including searchable
payload even when hidden from output, and query options bind the snapshot. Search/context rejects
summaries and handoff --at. None of these output limits bounds source scanning or runtime.

For cross-file text search with the same -B/-A/-C convention, load the `search` reference.

## Structured event ledger

Use `nn transcript events <session> <agent-id> --json` for deterministic event analysis instead
of downloading raw show payloads and writing another parser. `--select identity,usage,tools`
selects facets; available facets are identity/message/usage/tools/lifecycle (all by default,
identity always included). `--payload` opts into native payloads; `--event <event-id>` retrieves
one exact event while preserving its full-ledger ordinal. Unknown event IDs fail.

Pages expose `version`, `snapshot`, `page`, `pages`, `next_page`, `select`, `payload`, `schema`,
`detail_status`, `event_filter`, and `events`. Retrieve every page with the same options and the
page-1 `--snapshot`. Normal entries are directly usable event objects. An oversized event instead
has `event_id`, `ordinal`, `segment`, `segments`, and `text`: concatenate its ordered text fragments
and JSON-decode before interpreting or counting it. Pages including newline are at most 48,000 bytes.

Events retain `event_id`, `agent_id`, `ordinal`, `kind`, `timestamp`, `timestamp_source`, and `source`
(path, decoded-record ordinal, native record ID). IDs identify source positions, not immutable content.
Order is source-path/decoded-record/extraction order, not cross-source chronology. Missing timestamps
are null; native message timestamps can be millisecond numbers rather than RFC3339 strings.
Usage appears only on assistant message events, never extracted tool events: known component counts,
`known_total_tokens`, complete/partial/unavailable status, and nullable `total_tokens`/`context_tokens`.
Missing counters are null, not zero. Context is input plus cache-read counts when both are known.
Tool joins report matched/missing/ambiguous/unavailable rather than guessing duplicate or missing IDs.
Message/result text bytes and Unicode characters, and serialized argument/content bytes, are **not tokens**.

Pi shares show's authenticated selection and also exposes matching producer terminal records.
SDK file ownership is confined; Claude Code inline child execution remains unavailable. `detail_status`
is unavailable when there are no usable selected messages, even if terminal records exist. Snapshots
bind selected projection/options, not unselected payload or original-source completeness. Payload and
arguments are omitted by default. Use event-specific payload retrieval to inspect a standout; behavioral
claims still require complete relevant evidence, not usage magnitude, result size, or producer status.

## Recorded failures and timestamp windows

The message facet exposes nullable `record_timestamp`, `message_timestamp`, `stop_reason`, and
`error_message`, retaining native clock values and Pi camelCase / snake_case failure fields.
For failure-field aliases, the first well-typed non-null value wins (camelCase first), including an
explicit empty string; meaningful display and error selection use the same precedence.
Meaningful `show` includes an escaped `[failure stop_reason="..." error_message="..."]` marker
for assistant errors, aborts, or nonempty error messages—even for thinking-only/empty responses.
It still omits thinking and tool-result bodies. Ordinary meaningful text search is not an error
index: use `events --errors-only` to select failure records without searching payload text.
This selects assistant failure **message events** and explicitly erroneous **tool_result events**,
not the duplicate enclosing tool-result message, unknown flags, or messages merely mentioning errors.
Selection is independent of requested facets; joins retain their full-ledger meaning even if their
other endpoint is outside the window. An outside-window endpoint requires a separate unfiltered
`--event` request; absence from a window does not mean a missing join.

`--since` and `--until` are inclusive RFC3339 bounds; either can be used alone, and errors-only
can be combined with them. `--last N` returns the final N events *after* those filters in canonical
ascending ledger order; use it for a bounded “what just happened?” read without `--all` or external
JSON trimming. Its query receipt reports `requested_last`, matching and returned counts, and whether
older matching events exist. `--last` requires N > 0 and rejects with `--all`, `--event`, `--summary`,
or `--at`; normal facet selection and `--payload` remain available. Reversed/empty/invalid bounds
reject. Window/error flags reject with --event or any --summary. Filters preserve original event
IDs/ordinals and ledger order; never re-sort or renumber the selected result. Events with unknown/invalid clocks are excluded only when
a time bound is active. The `query` receipt reports normalized bounds, `errors_only`, clock/boundary,
full `ledger_snapshot`, total/selected event counts, `excluded_before`, `excluded_after`,
`excluded_unknown_timestamp`, `excluded_non_errors`, first/last selected event, and completeness
qualification. Exclusion counts form a partition: clock/window exclusion happens before error filtering.

Retrieve **every** page and ordered segment using unchanged filters and the page-1 `--snapshot`.
The snapshot binds the full evidence projection, query, and selected output; even an out-of-window
projected change rejects continuation. This establishes complete selected transport, not historical
source completeness. Empty windows remain valid receipts with zero selected events and null boundaries.
`--all` is an explicit unbounded complete filtered export with the same snapshot; normal pages remain
at most 48,000 bytes. Use exact event payload retrieval after locating an interval/error standout.

## Complete show retrieval

For an explicit unbounded JSON export, use `show --all --json` (complete text) or
`events --all` (complete event objects) under the export contract above. Otherwise:
Retrieve every page under the page-1 snapshot, concatenate `segments[].text` by global
`segment` ordinal, and verify all `segments` ordinals are present before interpreting events.
Never mix snapshots or make event-derived claims from a partial page set. The reconstruction is
exactly the legacy text `show` projection for the selected meaningful/raw mode.
The snapshot binds the request and projected output, not original source bytes. JSON decoding
may replace malformed source UTF-8 with U+FFFD; JSON pagination rejects invalid projected UTF-8.
Resolved Pi sidechain events require an explicit matching `agentId`, including in `--raw` mode;
foreign or missing-owner records are not attributable detail. If none match, metadata fallback
means event detail is unavailable, not that the agent did no work.
For Pi, raw detail contains complete owned message payloads, not outer JSONL wrappers.
Meaningful show and search omit tool-result roles and typed tool-result blocks; use `--raw`
when inspecting or locating tool-result errors or payloads.


