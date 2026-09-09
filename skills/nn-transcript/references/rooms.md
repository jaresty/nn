---
name: rooms
applies_when: "When entering one authenticated agent room, rearranging its Situation Board, inspecting room evidence, expanding its recent window, refreshing it, or returning to its source Office Scan."
---

# nn-transcript / rooms — Situation Board

A room is the analytical destination, not a continuation of the office metaphor. On entry, show a
**Situation Board**: a bounded, evidence-backed spatial projection for one authenticated agent.
Load **events** before event retrieval, **handoffs** before lifecycle claims, **summaries** before
usage/tool/timing reductions, and **lenses** for shared projection rules.

## Initial room entry

Retrieve a metadata-oriented bounded tail:

```bash
nn transcript events <session> <agent-id> --last 5 --json
```

State the room identity, snapshot, matching and returned event counts, and whether older matching
events exist. Initial entry shows a **neutral five-event orientation** in canonical ledger order; it
**does not choose a lens**, infer salient dimensions, or select axes unless the entering request also
said `orient me`, `choose for me`, or supplied an explicit lens. Then offer lens choices. Do not request
payloads by default. Inspect an exact event with `--event <event-id> --payload` when its content matters.
A comparison request lacking **comparison operands** opens a chooser rather than inventing a comparison
set.

After a lens is selected, state the active lens, declared axes, selected filters, and evidence boundary.
The default board may use canonical ledger order × evidence kind, but user-defined arrangements are
first-class and named lenses are presets. The human may supply arbitrary questions, axes, grouping,
filters, comparisons, or visual metaphors. Apply only dimensions supported by retrieved evidence and
label interpreted dimensions.

## Situation Board grammar

Keep the room identity separate from plotted findings. Declare both axes and every mark. Useful
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

Every Situation Board and selected-event detail view visibly offers **Scan this level**, **Change lens**,
**Back**, and **End** alongside context-specific actions such as inspect event or expand recent window.
At room scope, Scan this level rearranges the retained event population; at **selected-event** scope,
it scans or compares the selected evidence without silently widening to the whole room. Controls are
plain-language affordances, not hidden colon commands.

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

[Inspect event 216] [Expand recent window] [Scan this level] [Change lens] [Back] [End]
```
