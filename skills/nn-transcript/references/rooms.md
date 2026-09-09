---
name: rooms
applies_when: "When entering one authenticated agent room, rearranging its Situation Board, inspecting room evidence, expanding its recent window, refreshing it, or returning to its source Office Scan."
---

# nn-transcript / rooms — Situation Board

A room is the analytical destination, not a continuation of the office metaphor. Its **Situation Board**
starts as a neutral readable orientation; a requested/delegated lens may make it spatial.
Load `nn skills get nn-transcript --reference interaction` for target resolution, retained views,
inspection envelopes, and Back; this reference owns room-entry presentation.
Load **events** before event retrieval, **handoffs** before lifecycle claims, **summaries** before
usage/tool/timing reductions, and **lenses** for shared projection rules.

## Attention evidence

Load **attention** with `nn skills get nn-transcript --reference attention` when entering via a signal
or requesting attention for this room. The target is this exact room, not siblings; the owner governs
retained display versus newly authorized evaluation. Show policy identity, ratio, counts, window, and
limitations; bind **Inspect evidence** to the exact retained attention snapshot and room, not a fresh tail.

## Initial room entry

Choose the initial evidence from the active question/action. For assignment alignment, load **context**
and use `nn transcript context <session> <agent-id> --last 5 --format text` first, budgeting assignments
as initial evidence. Do not always infer alignment from 'inspect recent work'. For neutral activity
orientation, retrieve a bounded readable tail within the inspection envelope:

```bash
nn transcript events <session> <agent-id> --last 5 --format text --max-text-chars 1000
```

State the room identity, snapshot, matching and returned event counts, and whether older matching
events exist. Initial entry shows a **neutral five-event orientation** in canonical ledger order; it
**does not choose a lens**, infer salient dimensions, or select axes unless the entering request also
said `orient me`, `choose for me`, or supplied an explicit lens. State what the visible content supports
and its truncation limits, then offer a concrete useful next action; do not require another 'orient me'
turn or a lens chooser. Inspect an exact event with `--event <event-id> --payload` when its content
matters and the envelope permits. A comparison lacking **comparison operands** in both the request
and a displayed bound action opens a chooser; do not invent its comparison set.

After a lens is selected, state the active lens, declared axes, selected filters, and evidence boundary.
The default board may use canonical ledger order × evidence kind, but user-defined arrangements are
first-class and named lenses are presets. The human may supply arbitrary questions, axes, grouping,
filters, comparisons, or visual metaphors. Apply only dimensions supported by retrieved evidence and
label interpreted dimensions.

## Situation Board grammar

When rendering a spatial lens, keep the room identity separate from plotted findings. Declare both axes and every mark. Useful
presets include timeline, evidence strength, failure surface, handoffs, work products, cost, timing,
claim map, and explicit dependencies, but they are not exhaustive.

Use claim-level evidence labels:

- `[M]` metadata only;
- `[R]` agent reported;
- `[I]` inspected retained result;
- `[V]` independent verification.

Never assign one evidence level to the whole room. Distinguish an agent report, an inspected retained
result, and independent verification. Icons indicate observed evidence kind, not success. Unknown or
uninspected findings remain visible. A recorded interruption does not establish its cause.

## Discoverable controls

Situation Board fallback shortcuts are `Orient me`, `Choose lens…`, `Inspect event…`, and `More…`.
Promote stronger evidence-based next actions using the core's suggested-action contract; displaced
controls remain under More. Capture is always available there or via “capture that”; promote
`Capture this insight` only for a useful supported candidate, opening an approval proposal.
Every **selected-event** detail defaults to `Explain`, `Compare…`, `Inspect payload`, and `More…`
shortcuts. `Back` and `End` remain visible at both levels. `More…` exposes uncommon operations such as
**Scan this level…**, **Change lens…**, expand recent window, or advanced comparison without inserting
another intermediate menu. At room scope a scan rearranges the retained event population; at selected-
event scope it scans or compares selected evidence without silently widening to the whole room.
Controls are plain-language affordances, not hidden colon commands.

## Conversational rearrangement

Permit “flip the axes,” “focus failures,” “compare reports with results,” “make unknowns prominent,”
“show another useful view,” or a completely user-defined lens. Before redrawing, state the active
scope, question, axes, grouping, filters, and evidence boundary. Preserve selected event IDs and
claim qualifications across rearrangements.

## Expand recent window

Offer **Expand recent window**, not “older page” or stable continuation:

```bash
nn transcript events <session> <agent-id> --last 20 --json
```

This creates a **replacement snapshot**. Say that it replaces the prior `--last 5` view and is
**not stable backward continuation** before the earlier boundary. Do not silently increase N or imply that
omitted earlier events do not exist.

## Refresh, detail, and return

An evidence-detail view preserves the active room arrangement on Back. A refresh **retains the lens**,
room identity, question, and filters, but **reacquires** mutable evidence. A discussion-only return
may reuse a **retained snapshot** if it is labeled as not refreshed.

When this room came from an Office Scan, Back restores that exact scan and its scope. Room detail may
correct the office-level impression; keep the scan's earlier claim as bounded prior context rather
than silently rewriting its provenance.

## Example

```text
SITUATION BOARD — agent 39e8…
View: interruption analysis
X: canonical ledger order
Y: evidence strength
Window: last 5 matching events; older matching events exist

[V] independently verified     ·
[I] inspected result           nearby focused test passed
[R] agent report               no causal explanation retained
[M] metadata                   ⚠ event 216: terminated
                               ───────────────────────→ ledger

Established: interruption record exists.
Not established: retry cause, provider latency, task failure, or current activity.

[Orient me] [Choose lens…] [Inspect event…] [More…] [Back] [End]
```
