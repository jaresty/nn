# ADR-0049: Event search and grep-style context windows

Status: Accepted for implementation

## Decision

Extend `nn transcript events` with `--kind`, `--role`, `--search`, explicit `--regex`,
`-B/--before-context`, `-A/--after-context`, and `-C/--context`. Existing invocations retain
existing output and selection semantics. Exact `--event` anchors also support readable text/context.

Filters combine with AND and select anchors, not context. Search is case-insensitive literal by
default; regex uses Go syntax and is case-sensitive unless the pattern requests otherwise. Search
covers readable message text, tool names/arguments/results and lifecycle status, never reasoning,
signatures or unrelated message metadata. Role selects message events only. Kind is one of message,
tool_call, tool_result, lifecycle. Exact ID cannot combine with anchor filters.

Context counts canonical ledger events, not messages/turns or elapsed time. Expand anchors within the
selected agent's full ledger, including surrounding events that fail the filters. Merge overlapping
or adjacent windows, retain original IDs/ordinals/joins, and mark matches versus context. Disclose gaps,
match limits and omitted events. Missing exact anchors fail rather than substituting a nearby event.

Filtered queries default to the first 20 matching anchors; `--max-matches` allows 1–200. `--last N`
selects recent matching anchors before applying that cap. Context bounds are 0–200; reject -C with -A/-B
rather than silently choosing precedence. At most 2,000 expanded events; reject larger windows with
explicit guidance. Text output is limited to 200,000 bytes and per-event clipping remains explicit.
These are output/selection limits, not source-read/runtime guarantees.

New query metadata and the full input projection bind the snapshot, including search inputs even
when payload output is suppressed. JSON keeps existing 48KB pagination/segmentation. Search metadata
is opt-in and old receipts remain unchanged. Context may cross time/error filters because those filters
select anchors; disclose this rule. New filtering/context is incompatible with summaries and handoffs.

## Search consistency

`transcript search` also accepts the same -B/-A/-C event-count bounds. Its existing whole-message
matching semantics, native event IDs, source ordering and global match limit remain unchanged.
Context adds a canonical message-event anchor ID and merges within each source/agent. Matched
message bytes are checked during context acquisition; missing ownership/changed anchors fail.
Context is a fresh read, not an atomic snapshot of initial multi-file search. The default 50-match
limit remains, capped at 200 with context. Context JSON/text is a lossy display, not payload transport;
1,000-character excerpts/event text, 2,000 expanded events globally, 1 MiB JSON or 200,000 text bytes.
Explicit omission counts accompany clipping; oversized responses fail before publication. Without
context flags, old output and bounds remain unchanged.

## Rationale

Questions such as “what did the agent say after running bar?” should not require raw-message searches
that expose opaque reasoning signatures, timestamp reconstruction, or manual whole-transcript slicing.
One event-relative interface covers locating commands and inspecting subsequent agent messages.

## Verification

Test filter composition, literal/regex distinctions, non-searchable reasoning/metadata, stable anchor
IDs, context beyond filtered kinds, overlapping windows/gaps, boundaries, absent anchors, bounds,
payload suppression, stale snapshots, pagination, adapter ownership, and unchanged legacy behavior.
Exercise the installed command against recorded work separately from fixture assertions.
